package state

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"

	_ "github.com/lib/pq"

	"github.com/danpasecinic/podling/internal/types"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// PostgresStore is a PostgreSQL implementation of StateStore
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL state store
func NewPostgresStore(connectionString string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &PostgresStore{db: db}

	if err := store.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

// Close closes the database connection
func (s *PostgresStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// runMigrations applies database schema using goose
func (s *PostgresStore) runMigrations() error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(s.db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// AddTask adds a new task to the store
func (s *PostgresStore) AddTask(task types.Task) error {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM tasks WHERE task_id = $1)", task.TaskID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check task existence: %w", err)
	}
	if exists {
		return ErrTaskAlreadyExists
	}

	envJSON, err := json.Marshal(task.Env)
	if err != nil {
		return fmt.Errorf("failed to marshal env: %w", err)
	}

	var livenessProbeJSON, readinessProbeJSON []byte
	if task.LivenessProbe != nil {
		livenessProbeJSON, err = json.Marshal(task.LivenessProbe)
		if err != nil {
			return fmt.Errorf("failed to marshal liveness probe: %w", err)
		}
	}
	if task.ReadinessProbe != nil {
		readinessProbeJSON, err = json.Marshal(task.ReadinessProbe)
		if err != nil {
			return fmt.Errorf("failed to marshal readiness probe: %w", err)
		}
	}

	query := `
		INSERT INTO tasks (task_id, name, image, env, status, node_id, container_id, created_at, started_at, finished_at, error, liveness_probe, readiness_probe, restart_policy, health_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err = s.db.Exec(
		query,
		task.TaskID,
		task.Name,
		task.Image,
		envJSON,
		task.Status,
		nullString(task.NodeID),
		nullString(task.ContainerID),
		task.CreatedAt,
		task.StartedAt,
		task.FinishedAt,
		nullString(task.Error),
		nullBytes(livenessProbeJSON),
		nullBytes(readinessProbeJSON),
		nullString(string(task.RestartPolicy)),
		nullString(string(task.HealthStatus)),
	)

	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	return nil
}

// GetTask retrieves a task by ID
func (s *PostgresStore) GetTask(taskID string) (types.Task, error) {
	query := `
		SELECT task_id, name, image, env, status, node_id, container_id, created_at, started_at, finished_at, error,
		       liveness_probe, readiness_probe, restart_policy, health_status
		FROM tasks
		WHERE task_id = $1
	`

	var task types.Task
	var envJSON, livenessProbeJSON, readinessProbeJSON []byte
	var nodeID, containerID, errorMsg, restartPolicy, healthStatus sql.NullString

	err := s.db.QueryRow(query, taskID).Scan(
		&task.TaskID,
		&task.Name,
		&task.Image,
		&envJSON,
		&task.Status,
		&nodeID,
		&containerID,
		&task.CreatedAt,
		&task.StartedAt,
		&task.FinishedAt,
		&errorMsg,
		&livenessProbeJSON,
		&readinessProbeJSON,
		&restartPolicy,
		&healthStatus,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return types.Task{}, ErrTaskNotFound
	}
	if err != nil {
		return types.Task{}, fmt.Errorf("failed to get task: %w", err)
	}

	if len(envJSON) > 0 {
		if err := json.Unmarshal(envJSON, &task.Env); err != nil {
			return types.Task{}, fmt.Errorf("failed to unmarshal env: %w", err)
		}
	}

	if len(livenessProbeJSON) > 0 {
		task.LivenessProbe = &types.HealthCheck{}
		if err := json.Unmarshal(livenessProbeJSON, task.LivenessProbe); err != nil {
			return types.Task{}, fmt.Errorf("failed to unmarshal liveness probe: %w", err)
		}
	}

	if len(readinessProbeJSON) > 0 {
		task.ReadinessProbe = &types.HealthCheck{}
		if err := json.Unmarshal(readinessProbeJSON, task.ReadinessProbe); err != nil {
			return types.Task{}, fmt.Errorf("failed to unmarshal readiness probe: %w", err)
		}
	}

	task.NodeID = nodeID.String
	task.ContainerID = containerID.String
	task.Error = errorMsg.String
	if restartPolicy.Valid {
		task.RestartPolicy = types.RestartPolicy(restartPolicy.String)
	}
	if healthStatus.Valid {
		task.HealthStatus = types.HealthStatus(healthStatus.String)
	}

	return task, nil
}

// UpdateTask updates specific fields of a task
func (s *PostgresStore) UpdateTask(taskID string, updates TaskUpdate) error {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM tasks WHERE task_id = $1)", taskID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check task existence: %w", err)
	}
	if !exists {
		return ErrTaskNotFound
	}

	query := "UPDATE tasks SET "
	var args []interface{}
	argPos := 1

	if updates.Status != nil {
		query += fmt.Sprintf("status = $%d, ", argPos)
		args = append(args, *updates.Status)
		argPos++
	}
	if updates.NodeID != nil {
		query += fmt.Sprintf("node_id = $%d, ", argPos)
		args = append(args, *updates.NodeID)
		argPos++
	}
	if updates.ContainerID != nil {
		query += fmt.Sprintf("container_id = $%d, ", argPos)
		args = append(args, *updates.ContainerID)
		argPos++
	}
	if updates.StartedAt != nil {
		query += fmt.Sprintf("started_at = $%d, ", argPos)
		args = append(args, *updates.StartedAt)
		argPos++
	}
	if updates.FinishedAt != nil {
		query += fmt.Sprintf("finished_at = $%d, ", argPos)
		args = append(args, *updates.FinishedAt)
		argPos++
	}
	if updates.Error != nil {
		query += fmt.Sprintf("error = $%d, ", argPos)
		args = append(args, *updates.Error)
		argPos++
	}
	if updates.HealthStatus != nil {
		query += fmt.Sprintf("health_status = $%d, ", argPos)
		args = append(args, *updates.HealthStatus)
		argPos++
	}

	query = query[:len(query)-2]
	query += fmt.Sprintf(" WHERE task_id = $%d", argPos)
	args = append(args, taskID)

	_, err = s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// ListTasks returns all tasks in the store
func (s *PostgresStore) ListTasks() ([]types.Task, error) {
	query := `
		SELECT task_id, name, image, env, status, node_id, container_id, created_at, started_at, finished_at, error,
		       liveness_probe, readiness_probe, restart_policy, health_status
		FROM tasks
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var tasks []types.Task
	for rows.Next() {
		var task types.Task
		var envJSON, livenessProbeJSON, readinessProbeJSON []byte
		var nodeID, containerID, errorMsg, restartPolicy, healthStatus sql.NullString

		err := rows.Scan(
			&task.TaskID,
			&task.Name,
			&task.Image,
			&envJSON,
			&task.Status,
			&nodeID,
			&containerID,
			&task.CreatedAt,
			&task.StartedAt,
			&task.FinishedAt,
			&errorMsg,
			&livenessProbeJSON,
			&readinessProbeJSON,
			&restartPolicy,
			&healthStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if len(envJSON) > 0 {
			if err := json.Unmarshal(envJSON, &task.Env); err != nil {
				return nil, fmt.Errorf("failed to unmarshal env: %w", err)
			}
		}

		if len(livenessProbeJSON) > 0 {
			task.LivenessProbe = &types.HealthCheck{}
			if err := json.Unmarshal(livenessProbeJSON, task.LivenessProbe); err != nil {
				return nil, fmt.Errorf("failed to unmarshal liveness probe: %w", err)
			}
		}

		if len(readinessProbeJSON) > 0 {
			task.ReadinessProbe = &types.HealthCheck{}
			if err := json.Unmarshal(readinessProbeJSON, task.ReadinessProbe); err != nil {
				return nil, fmt.Errorf("failed to unmarshal readiness probe: %w", err)
			}
		}

		task.NodeID = nodeID.String
		task.ContainerID = containerID.String
		task.Error = errorMsg.String
		if restartPolicy.Valid {
			task.RestartPolicy = types.RestartPolicy(restartPolicy.String)
		}
		if healthStatus.Valid {
			task.HealthStatus = types.HealthStatus(healthStatus.String)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// AddPod adds a new pod to the store
func (s *PostgresStore) AddPod(pod types.Pod) error {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM pods WHERE pod_id = $1)", pod.PodID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check pod existence: %w", err)
	}
	if exists {
		return ErrPodAlreadyExists
	}

	labelsJSON, err := json.Marshal(pod.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	annotationsJSON, err := json.Marshal(pod.Annotations)
	if err != nil {
		return fmt.Errorf("failed to marshal annotations: %w", err)
	}

	containersJSON, err := json.Marshal(pod.Containers)
	if err != nil {
		return fmt.Errorf("failed to marshal containers: %w", err)
	}

	query := `
		INSERT INTO pods (pod_id, name, namespace, labels, annotations, containers, status, node_id, restart_policy, created_at, scheduled_at, started_at, finished_at, message, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err = s.db.Exec(
		query,
		pod.PodID,
		pod.Name,
		nullString(pod.Namespace),
		labelsJSON,
		annotationsJSON,
		containersJSON,
		pod.Status,
		nullString(pod.NodeID),
		nullString(string(pod.RestartPolicy)),
		pod.CreatedAt,
		pod.ScheduledAt,
		pod.StartedAt,
		pod.FinishedAt,
		nullString(pod.Message),
		nullString(pod.Reason),
	)

	if err != nil {
		return fmt.Errorf("failed to insert pod: %w", err)
	}

	return nil
}

// GetPod retrieves a pod by ID
func (s *PostgresStore) GetPod(podID string) (types.Pod, error) {
	query := `
		SELECT pod_id, name, namespace, labels, annotations, containers, status, node_id, restart_policy, created_at, scheduled_at, started_at, finished_at, message, reason
		FROM pods
		WHERE pod_id = $1
	`

	var pod types.Pod
	var labelsJSON, annotationsJSON, containersJSON []byte
	var namespace, nodeID, restartPolicy, message, reason sql.NullString

	err := s.db.QueryRow(query, podID).Scan(
		&pod.PodID,
		&pod.Name,
		&namespace,
		&labelsJSON,
		&annotationsJSON,
		&containersJSON,
		&pod.Status,
		&nodeID,
		&restartPolicy,
		&pod.CreatedAt,
		&pod.ScheduledAt,
		&pod.StartedAt,
		&pod.FinishedAt,
		&message,
		&reason,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return types.Pod{}, ErrPodNotFound
	}
	if err != nil {
		return types.Pod{}, fmt.Errorf("failed to get pod: %w", err)
	}

	if len(labelsJSON) > 0 {
		if err := json.Unmarshal(labelsJSON, &pod.Labels); err != nil {
			return types.Pod{}, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
	}

	pod.Annotations = make(map[string]string)
	if len(annotationsJSON) > 0 {
		var testArray []interface{}
		if json.Unmarshal(annotationsJSON, &testArray) == nil {
			pod.Annotations = make(map[string]string)
		} else {
			if err := json.Unmarshal(annotationsJSON, &pod.Annotations); err != nil {
				return types.Pod{}, fmt.Errorf("failed to unmarshal annotations: %w", err)
			}
		}
	}

	if err := json.Unmarshal(containersJSON, &pod.Containers); err != nil {
		return types.Pod{}, fmt.Errorf("failed to unmarshal containers: %w", err)
	}

	pod.Namespace = namespace.String
	pod.NodeID = nodeID.String
	pod.Message = message.String
	pod.Reason = reason.String
	if restartPolicy.Valid {
		pod.RestartPolicy = types.RestartPolicy(restartPolicy.String)
	}

	return pod, nil
}

// UpdatePod updates specific fields of a pod
func (s *PostgresStore) UpdatePod(podID string, updates PodUpdate) error {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM pods WHERE pod_id = $1)", podID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check pod existence: %w", err)
	}
	if !exists {
		return ErrPodNotFound
	}

	query := "UPDATE pods SET "
	var args []interface{}
	argPos := 1

	if updates.Status != nil {
		query += fmt.Sprintf("status = $%d, ", argPos)
		args = append(args, *updates.Status)
		argPos++
	}
	if updates.NodeID != nil {
		query += fmt.Sprintf("node_id = $%d, ", argPos)
		args = append(args, *updates.NodeID)
		argPos++
	}
	if updates.Containers != nil {
		containersJSON, err := json.Marshal(updates.Containers)
		if err != nil {
			return fmt.Errorf("failed to marshal containers: %w", err)
		}
		query += fmt.Sprintf("containers = $%d, ", argPos)
		args = append(args, containersJSON)
		argPos++
	}
	if updates.ScheduledAt != nil {
		query += fmt.Sprintf("scheduled_at = $%d, ", argPos)
		args = append(args, *updates.ScheduledAt)
		argPos++
	}
	if updates.StartedAt != nil {
		query += fmt.Sprintf("started_at = $%d, ", argPos)
		args = append(args, *updates.StartedAt)
		argPos++
	}
	if updates.FinishedAt != nil {
		query += fmt.Sprintf("finished_at = $%d, ", argPos)
		args = append(args, *updates.FinishedAt)
		argPos++
	}
	if updates.Message != nil {
		query += fmt.Sprintf("message = $%d, ", argPos)
		args = append(args, *updates.Message)
		argPos++
	}
	if updates.Reason != nil {
		query += fmt.Sprintf("reason = $%d, ", argPos)
		args = append(args, *updates.Reason)
		argPos++
	}
	if updates.Annotations != nil {
		// For PostgreSQL, we merge annotations using JSONB concatenation
		annotationsJSON, err := json.Marshal(*updates.Annotations)
		if err != nil {
			return fmt.Errorf("failed to marshal annotations: %w", err)
		}
		query += fmt.Sprintf(
			"annotations = COALESCE(CASE WHEN jsonb_typeof(annotations) = 'array' THEN '{}'::jsonb ELSE annotations END, '{}'::jsonb) || $%d, ",
			argPos,
		)
		args = append(args, annotationsJSON)
		argPos++
	}

	query = query[:len(query)-2]
	query += fmt.Sprintf(" WHERE pod_id = $%d", argPos)
	args = append(args, podID)

	_, err = s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update pod: %w", err)
	}

	return nil
}

// ListPods returns all pods in the store
func (s *PostgresStore) ListPods() ([]types.Pod, error) {
	query := `
		SELECT pod_id, name, namespace, labels, annotations, containers, status, node_id, restart_policy, created_at, scheduled_at, started_at, finished_at, message, reason
		FROM pods
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pods: %w", err)
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var pods []types.Pod
	for rows.Next() {
		var pod types.Pod
		var labelsJSON, annotationsJSON, containersJSON []byte
		var namespace, nodeID, restartPolicy, message, reason sql.NullString

		err := rows.Scan(
			&pod.PodID,
			&pod.Name,
			&namespace,
			&labelsJSON,
			&annotationsJSON,
			&containersJSON,
			&pod.Status,
			&nodeID,
			&restartPolicy,
			&pod.CreatedAt,
			&pod.ScheduledAt,
			&pod.StartedAt,
			&pod.FinishedAt,
			&message,
			&reason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pod: %w", err)
		}

		if len(labelsJSON) > 0 && string(labelsJSON) != "null" {
			pod.Labels = make(map[string]string)
			if err := json.Unmarshal(labelsJSON, &pod.Labels); err != nil {
				return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
			}
		} else {
			pod.Labels = make(map[string]string)
		}

		pod.Annotations = make(map[string]string)
		if len(annotationsJSON) > 0 && string(annotationsJSON) != "null" {
			var testArray []interface{}
			if json.Unmarshal(annotationsJSON, &testArray) == nil {
				pod.Annotations = make(map[string]string)
			} else {
				if err := json.Unmarshal(annotationsJSON, &pod.Annotations); err != nil {
					return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
				}
			}
		}

		if err := json.Unmarshal(containersJSON, &pod.Containers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal containers: %w", err)
		}

		pod.Namespace = namespace.String
		pod.NodeID = nodeID.String
		pod.Message = message.String
		pod.Reason = reason.String
		if restartPolicy.Valid {
			pod.RestartPolicy = types.RestartPolicy(restartPolicy.String)
		}

		pods = append(pods, pod)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pods: %w", err)
	}

	return pods, nil
}

// DeletePod removes a pod from the store
func (s *PostgresStore) DeletePod(podID string) error {
	result, err := s.db.Exec("DELETE FROM pods WHERE pod_id = $1", podID)
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPodNotFound
	}

	return nil
}

// AddNode adds a new node to the store
func (s *PostgresStore) AddNode(node types.Node) error {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM nodes WHERE node_id = $1)", node.NodeID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check node existence: %w", err)
	}
	if exists {
		return ErrNodeAlreadyExists
	}

	query := `
		INSERT INTO nodes (node_id, hostname, port, status, running_tasks, last_heartbeat, resources)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	resourcesJSON, err := json.Marshal(node.Resources)
	if err != nil {
		return fmt.Errorf("failed to marshal resources: %w", err)
	}

	_, err = s.db.Exec(
		query,
		node.NodeID,
		node.Hostname,
		node.Port,
		node.Status,
		node.RunningTasks,
		node.LastHeartbeat,
		resourcesJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert node: %w", err)
	}

	return nil
}

// GetNode retrieves a node by ID
func (s *PostgresStore) GetNode(nodeID string) (types.Node, error) {
	query := `
		SELECT node_id, hostname, port, status, running_tasks, last_heartbeat, resources
		FROM nodes
		WHERE node_id = $1
	`

	var node types.Node
	var resourcesJSON []byte
	err := s.db.QueryRow(query, nodeID).Scan(
		&node.NodeID,
		&node.Hostname,
		&node.Port,
		&node.Status,
		&node.RunningTasks,
		&node.LastHeartbeat,
		&resourcesJSON,
	)

	if err == nil {
		if err := json.Unmarshal(resourcesJSON, &node.Resources); err != nil {
			return types.Node{}, fmt.Errorf("failed to unmarshal resources: %w", err)
		}
	}

	if errors.Is(err, sql.ErrNoRows) {
		return types.Node{}, ErrNodeNotFound
	}
	if err != nil {
		return types.Node{}, fmt.Errorf("failed to get node: %w", err)
	}

	return node, nil
}

// UpdateNode updates specific fields of a node
func (s *PostgresStore) UpdateNode(nodeID string, updates NodeUpdate) error {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM nodes WHERE node_id = $1)", nodeID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check node existence: %w", err)
	}
	if !exists {
		return ErrNodeNotFound
	}

	query := "UPDATE nodes SET "
	var args []interface{}
	argPos := 1

	if updates.Status != nil {
		query += fmt.Sprintf("status = $%d, ", argPos)
		args = append(args, *updates.Status)
		argPos++
	}
	if updates.RunningTasks != nil {
		query += fmt.Sprintf("running_tasks = $%d, ", argPos)
		args = append(args, *updates.RunningTasks)
		argPos++
	}
	if updates.LastHeartbeat != nil {
		query += fmt.Sprintf("last_heartbeat = $%d, ", argPos)
		args = append(args, *updates.LastHeartbeat)
		argPos++
	}

	query = query[:len(query)-2]
	query += fmt.Sprintf(" WHERE node_id = $%d", argPos)
	args = append(args, nodeID)

	_, err = s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	return nil
}

// ListNodes returns all nodes in the store
func (s *PostgresStore) ListNodes() ([]types.Node, error) {
	query := `
		SELECT node_id, hostname, port, status, running_tasks, last_heartbeat, resources
		FROM nodes
		ORDER BY last_heartbeat DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var nodes []types.Node
	for rows.Next() {
		var node types.Node
		var resourcesJSON []byte
		err := rows.Scan(
			&node.NodeID,
			&node.Hostname,
			&node.Port,
			&node.Status,
			&node.RunningTasks,
			&node.LastHeartbeat,
			&resourcesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		if err := json.Unmarshal(resourcesJSON, &node.Resources); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}

		nodes = append(nodes, node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nodes: %w", err)
	}

	return nodes, nil
}

// GetAvailableNodes returns all online nodes with available capacity
func (s *PostgresStore) GetAvailableNodes() ([]types.Node, error) {
	query := `
		SELECT node_id, hostname, port, status, running_tasks, last_heartbeat, resources
		FROM nodes
		WHERE status = $1
		ORDER BY running_tasks ASC
	`

	rows, err := s.db.Query(query, types.NodeOnline)
	if err != nil {
		return nil, fmt.Errorf("failed to query available nodes: %w", err)
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var nodes []types.Node
	for rows.Next() {
		var node types.Node
		var resourcesJSON []byte
		err := rows.Scan(
			&node.NodeID,
			&node.Hostname,
			&node.Port,
			&node.Status,
			&node.RunningTasks,
			&node.LastHeartbeat,
			&resourcesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		if err := json.Unmarshal(resourcesJSON, &node.Resources); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}

		// Filter nodes that have available capacity
		maxSlots := node.GetMaxTaskSlots()
		if node.RunningTasks >= maxSlots {
			continue
		}

		nodes = append(nodes, node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nodes: %w", err)
	}

	return nodes, nil
}

// nullString converts an empty string to sql.NullString
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullBytes(b []byte) interface{} {
	if b == nil {
		return nil
	}
	return b
}

// AddService adds a new service to the store
func (s *PostgresStore) AddService(service types.Service) error {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM services WHERE service_id = $1)", service.ServiceID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check service existence: %w", err)
	}
	if exists {
		return ErrServiceAlreadyExists
	}

	selectorJSON, err := json.Marshal(service.Selector)
	if err != nil {
		return fmt.Errorf("failed to marshal selector: %w", err)
	}

	portsJSON, err := json.Marshal(service.Ports)
	if err != nil {
		return fmt.Errorf("failed to marshal ports: %w", err)
	}

	labelsJSON, err := json.Marshal(service.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	annotationsJSON, err := json.Marshal(service.Annotations)
	if err != nil {
		return fmt.Errorf("failed to marshal annotations: %w", err)
	}

	query := `
		INSERT INTO services (service_id, name, namespace, type, cluster_ip, selector, ports, labels, annotations, session_affinity, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = s.db.Exec(
		query,
		service.ServiceID,
		service.Name,
		nullString(service.Namespace),
		service.Type,
		nullString(service.ClusterIP),
		selectorJSON,
		portsJSON,
		labelsJSON,
		annotationsJSON,
		nullString(service.SessionAffinity),
		service.CreatedAt,
		service.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert service: %w", err)
	}

	return nil
}

// GetService retrieves a service by ID
func (s *PostgresStore) GetService(serviceID string) (types.Service, error) {
	query := `
		SELECT service_id, name, namespace, type, cluster_ip, selector, ports, labels, annotations, session_affinity, created_at, updated_at
		FROM services
		WHERE service_id = $1
	`

	var service types.Service
	var namespace, clusterIP, sessionAffinity sql.NullString
	var selectorJSON, portsJSON, labelsJSON, annotationsJSON []byte

	err := s.db.QueryRow(query, serviceID).Scan(
		&service.ServiceID,
		&service.Name,
		&namespace,
		&service.Type,
		&clusterIP,
		&selectorJSON,
		&portsJSON,
		&labelsJSON,
		&annotationsJSON,
		&sessionAffinity,
		&service.CreatedAt,
		&service.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Service{}, ErrServiceNotFound
		}
		return types.Service{}, fmt.Errorf("failed to query service: %w", err)
	}

	if namespace.Valid {
		service.Namespace = namespace.String
	}
	if clusterIP.Valid {
		service.ClusterIP = clusterIP.String
	}
	if sessionAffinity.Valid {
		service.SessionAffinity = sessionAffinity.String
	}

	if err := json.Unmarshal(selectorJSON, &service.Selector); err != nil {
		return types.Service{}, fmt.Errorf("failed to unmarshal selector: %w", err)
	}
	if err := json.Unmarshal(portsJSON, &service.Ports); err != nil {
		return types.Service{}, fmt.Errorf("failed to unmarshal ports: %w", err)
	}
	if err := json.Unmarshal(labelsJSON, &service.Labels); err != nil {
		return types.Service{}, fmt.Errorf("failed to unmarshal labels: %w", err)
	}
	if err := json.Unmarshal(annotationsJSON, &service.Annotations); err != nil {
		return types.Service{}, fmt.Errorf("failed to unmarshal annotations: %w", err)
	}

	return service, nil
}

// GetServiceByName retrieves a service by namespace and name
func (s *PostgresStore) GetServiceByName(namespace, name string) (types.Service, error) {
	if namespace == "" {
		namespace = "default"
	}

	query := `
		SELECT service_id, name, namespace, type, cluster_ip, selector, ports, labels, annotations, session_affinity, created_at, updated_at
		FROM services
		WHERE COALESCE(namespace, 'default') = $1 AND name = $2
	`

	var service types.Service
	var ns, clusterIP, sessionAffinity sql.NullString
	var selectorJSON, portsJSON, labelsJSON, annotationsJSON []byte

	err := s.db.QueryRow(query, namespace, name).Scan(
		&service.ServiceID,
		&service.Name,
		&ns,
		&service.Type,
		&clusterIP,
		&selectorJSON,
		&portsJSON,
		&labelsJSON,
		&annotationsJSON,
		&sessionAffinity,
		&service.CreatedAt,
		&service.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Service{}, ErrServiceNotFound
		}
		return types.Service{}, fmt.Errorf("failed to query service: %w", err)
	}

	if ns.Valid {
		service.Namespace = ns.String
	}
	if clusterIP.Valid {
		service.ClusterIP = clusterIP.String
	}
	if sessionAffinity.Valid {
		service.SessionAffinity = sessionAffinity.String
	}

	if err := json.Unmarshal(selectorJSON, &service.Selector); err != nil {
		return types.Service{}, fmt.Errorf("failed to unmarshal selector: %w", err)
	}
	if err := json.Unmarshal(portsJSON, &service.Ports); err != nil {
		return types.Service{}, fmt.Errorf("failed to unmarshal ports: %w", err)
	}
	if err := json.Unmarshal(labelsJSON, &service.Labels); err != nil {
		return types.Service{}, fmt.Errorf("failed to unmarshal labels: %w", err)
	}
	if err := json.Unmarshal(annotationsJSON, &service.Annotations); err != nil {
		return types.Service{}, fmt.Errorf("failed to unmarshal annotations: %w", err)
	}

	return service, nil
}

// UpdateService updates specific fields of a service
func (s *PostgresStore) UpdateService(serviceID string, updates types.ServiceUpdate) error {
	// First check if service exists
	_, err := s.GetService(serviceID)
	if err != nil {
		return err
	}

	query := "UPDATE services SET updated_at = NOW()"
	args := []interface{}{}
	argNum := 1

	if updates.Selector != nil {
		selectorJSON, err := json.Marshal(*updates.Selector)
		if err != nil {
			return fmt.Errorf("failed to marshal selector: %w", err)
		}
		query += fmt.Sprintf(", selector = $%d", argNum)
		args = append(args, selectorJSON)
		argNum++
	}

	if updates.Ports != nil {
		portsJSON, err := json.Marshal(*updates.Ports)
		if err != nil {
			return fmt.Errorf("failed to marshal ports: %w", err)
		}
		query += fmt.Sprintf(", ports = $%d", argNum)
		args = append(args, portsJSON)
		argNum++
	}

	if updates.Labels != nil {
		labelsJSON, err := json.Marshal(*updates.Labels)
		if err != nil {
			return fmt.Errorf("failed to marshal labels: %w", err)
		}
		query += fmt.Sprintf(", labels = $%d", argNum)
		args = append(args, labelsJSON)
		argNum++
	}

	if updates.Annotations != nil {
		annotationsJSON, err := json.Marshal(*updates.Annotations)
		if err != nil {
			return fmt.Errorf("failed to marshal annotations: %w", err)
		}
		query += fmt.Sprintf(", annotations = $%d", argNum)
		args = append(args, annotationsJSON)
		argNum++
	}

	if updates.SessionAffinity != nil {
		query += fmt.Sprintf(", session_affinity = $%d", argNum)
		args = append(args, *updates.SessionAffinity)
		argNum++
	}

	query += fmt.Sprintf(" WHERE service_id = $%d", argNum)
	args = append(args, serviceID)

	_, err = s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	return nil
}

// ListServices returns all services in the specified namespace
func (s *PostgresStore) ListServices(namespace string) ([]types.Service, error) {
	var query string
	var args []interface{}

	if namespace == "" {
		query = `
			SELECT service_id, name, namespace, type, cluster_ip, selector, ports, labels, annotations, session_affinity, created_at, updated_at
			FROM services
			ORDER BY created_at DESC
		`
	} else {
		query = `
			SELECT service_id, name, namespace, type, cluster_ip, selector, ports, labels, annotations, session_affinity, created_at, updated_at
			FROM services
			WHERE COALESCE(namespace, 'default') = $1
			ORDER BY created_at DESC
		`
		args = append(args, namespace)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query services: %w", err)
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var services []types.Service
	for rows.Next() {
		var service types.Service
		var ns, clusterIP, sessionAffinity sql.NullString
		var selectorJSON, portsJSON, labelsJSON, annotationsJSON []byte

		err := rows.Scan(
			&service.ServiceID,
			&service.Name,
			&ns,
			&service.Type,
			&clusterIP,
			&selectorJSON,
			&portsJSON,
			&labelsJSON,
			&annotationsJSON,
			&sessionAffinity,
			&service.CreatedAt,
			&service.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}

		if ns.Valid {
			service.Namespace = ns.String
		}
		if clusterIP.Valid {
			service.ClusterIP = clusterIP.String
		}
		if sessionAffinity.Valid {
			service.SessionAffinity = sessionAffinity.String
		}

		if err := json.Unmarshal(selectorJSON, &service.Selector); err != nil {
			return nil, fmt.Errorf("failed to unmarshal selector: %w", err)
		}
		if err := json.Unmarshal(portsJSON, &service.Ports); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ports: %w", err)
		}
		if err := json.Unmarshal(labelsJSON, &service.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
		if err := json.Unmarshal(annotationsJSON, &service.Annotations); err != nil {
			return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
		}

		services = append(services, service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating services: %w", err)
	}

	return services, nil
}

// DeleteService removes a service from the store
func (s *PostgresStore) DeleteService(serviceID string) error {
	result, err := s.db.Exec("DELETE FROM services WHERE service_id = $1", serviceID)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrServiceNotFound
	}

	return nil
}

// SetEndpoints sets or updates endpoints for a service
func (s *PostgresStore) SetEndpoints(endpoints types.Endpoints) error {
	subsetsJSON, err := json.Marshal(endpoints.Subsets)
	if err != nil {
		return fmt.Errorf("failed to marshal subsets: %w", err)
	}

	query := `
		INSERT INTO endpoints (service_id, service_name, namespace, subsets, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (service_id) DO UPDATE
		SET subsets = EXCLUDED.subsets, updated_at = NOW()
	`

	_, err = s.db.Exec(
		query,
		endpoints.ServiceID,
		endpoints.ServiceName,
		nullString(endpoints.Namespace),
		subsetsJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert endpoints: %w", err)
	}

	return nil
}

// GetEndpoints retrieves endpoints by service ID
func (s *PostgresStore) GetEndpoints(serviceID string) (types.Endpoints, error) {
	query := `
		SELECT service_id, service_name, namespace, subsets, updated_at
		FROM endpoints
		WHERE service_id = $1
	`

	var endpoints types.Endpoints
	var namespace sql.NullString
	var subsetsJSON []byte

	err := s.db.QueryRow(query, serviceID).Scan(
		&endpoints.ServiceID,
		&endpoints.ServiceName,
		&namespace,
		&subsetsJSON,
		&endpoints.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Endpoints{}, ErrEndpointsNotFound
		}
		return types.Endpoints{}, fmt.Errorf("failed to query endpoints: %w", err)
	}

	if namespace.Valid {
		endpoints.Namespace = namespace.String
	}

	if err := json.Unmarshal(subsetsJSON, &endpoints.Subsets); err != nil {
		return types.Endpoints{}, fmt.Errorf("failed to unmarshal subsets: %w", err)
	}

	return endpoints, nil
}

// GetEndpointsByServiceName retrieves endpoints by namespace and service name
func (s *PostgresStore) GetEndpointsByServiceName(namespace, serviceName string) (types.Endpoints, error) {
	if namespace == "" {
		namespace = "default"
	}

	query := `
		SELECT service_id, service_name, namespace, subsets, updated_at
		FROM endpoints
		WHERE COALESCE(namespace, 'default') = $1 AND service_name = $2
	`

	var endpoints types.Endpoints
	var ns sql.NullString
	var subsetsJSON []byte

	err := s.db.QueryRow(query, namespace, serviceName).Scan(
		&endpoints.ServiceID,
		&endpoints.ServiceName,
		&ns,
		&subsetsJSON,
		&endpoints.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Endpoints{}, ErrEndpointsNotFound
		}
		return types.Endpoints{}, fmt.Errorf("failed to query endpoints: %w", err)
	}

	if ns.Valid {
		endpoints.Namespace = ns.String
	}

	if err := json.Unmarshal(subsetsJSON, &endpoints.Subsets); err != nil {
		return types.Endpoints{}, fmt.Errorf("failed to unmarshal subsets: %w", err)
	}

	return endpoints, nil
}

// DeleteEndpoints removes endpoints from the store
func (s *PostgresStore) DeleteEndpoints(serviceID string) error {
	result, err := s.db.Exec("DELETE FROM endpoints WHERE service_id = $1", serviceID)
	if err != nil {
		return fmt.Errorf("failed to delete endpoints: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrEndpointsNotFound
	}

	return nil
}

// ListPodsByLabels returns pods matching the label selector in a namespace
func (s *PostgresStore) ListPodsByLabels(namespace string, labels map[string]string) ([]types.Pod, error) {
	if namespace == "" {
		namespace = "default"
	}

	query := `
		SELECT pod_id, name, namespace, labels, annotations, containers, status, node_id, restart_policy, created_at, scheduled_at, started_at, finished_at, message, reason
		FROM pods
		WHERE COALESCE(namespace, 'default') = $1 AND labels @> $2
		ORDER BY created_at DESC
	`

	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal labels: %w", err)
	}

	rows, err := s.db.Query(query, namespace, labelsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to query pods by labels: %w", err)
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var pods []types.Pod
	for rows.Next() {
		var pod types.Pod
		var ns, nodeID, message, reason sql.NullString
		var scheduledAt, startedAt, finishedAt sql.NullTime
		var labelsJSON, annotationsJSON, containersJSON []byte

		err := rows.Scan(
			&pod.PodID,
			&pod.Name,
			&ns,
			&labelsJSON,
			&annotationsJSON,
			&containersJSON,
			&pod.Status,
			&nodeID,
			&pod.RestartPolicy,
			&pod.CreatedAt,
			&scheduledAt,
			&startedAt,
			&finishedAt,
			&message,
			&reason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pod: %w", err)
		}

		if ns.Valid {
			pod.Namespace = ns.String
		}
		if nodeID.Valid {
			pod.NodeID = nodeID.String
		}
		if message.Valid {
			pod.Message = message.String
		}
		if reason.Valid {
			pod.Reason = reason.String
		}
		if scheduledAt.Valid {
			pod.ScheduledAt = &scheduledAt.Time
		}
		if startedAt.Valid {
			pod.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			pod.FinishedAt = &finishedAt.Time
		}

		if err := json.Unmarshal(labelsJSON, &pod.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
		pod.Annotations = make(map[string]string)
		if len(annotationsJSON) > 0 {
			var testArray []interface{}
			if json.Unmarshal(annotationsJSON, &testArray) == nil {
				pod.Annotations = make(map[string]string)
			} else {
				if err := json.Unmarshal(annotationsJSON, &pod.Annotations); err != nil {
					return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
				}
			}
		}
		if err := json.Unmarshal(containersJSON, &pod.Containers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal containers: %w", err)
		}

		pods = append(pods, pod)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pods: %w", err)
	}

	return pods, nil
}
