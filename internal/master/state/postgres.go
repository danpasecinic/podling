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

	if len(annotationsJSON) > 0 {
		if err := json.Unmarshal(annotationsJSON, &pod.Annotations); err != nil {
			return types.Pod{}, fmt.Errorf("failed to unmarshal annotations: %w", err)
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

		if len(labelsJSON) > 0 {
			if err := json.Unmarshal(labelsJSON, &pod.Labels); err != nil {
				return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
			}
		}

		if len(annotationsJSON) > 0 {
			if err := json.Unmarshal(annotationsJSON, &pod.Annotations); err != nil {
				return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
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
		INSERT INTO nodes (node_id, hostname, port, status, capacity, running_tasks, last_heartbeat)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = s.db.Exec(
		query,
		node.NodeID,
		node.Hostname,
		node.Port,
		node.Status,
		node.Capacity,
		node.RunningTasks,
		node.LastHeartbeat,
	)

	if err != nil {
		return fmt.Errorf("failed to insert node: %w", err)
	}

	return nil
}

// GetNode retrieves a node by ID
func (s *PostgresStore) GetNode(nodeID string) (types.Node, error) {
	query := `
		SELECT node_id, hostname, port, status, capacity, running_tasks, last_heartbeat
		FROM nodes
		WHERE node_id = $1
	`

	var node types.Node
	err := s.db.QueryRow(query, nodeID).Scan(
		&node.NodeID,
		&node.Hostname,
		&node.Port,
		&node.Status,
		&node.Capacity,
		&node.RunningTasks,
		&node.LastHeartbeat,
	)

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
		SELECT node_id, hostname, port, status, capacity, running_tasks, last_heartbeat
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
		err := rows.Scan(
			&node.NodeID,
			&node.Hostname,
			&node.Port,
			&node.Status,
			&node.Capacity,
			&node.RunningTasks,
			&node.LastHeartbeat,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
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
		SELECT node_id, hostname, port, status, capacity, running_tasks, last_heartbeat
		FROM nodes
		WHERE status = $1 AND running_tasks < capacity
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
		err := rows.Scan(
			&node.NodeID,
			&node.Hostname,
			&node.Port,
			&node.Status,
			&node.Capacity,
			&node.RunningTasks,
			&node.LastHeartbeat,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
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
