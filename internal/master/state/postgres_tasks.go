package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/danpasecinic/podling/internal/types"
)

func (s *PostgresStateStore) AddTask(task types.Task) error {
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

	var portsJSON, resourcesJSON []byte
	if len(task.Ports) > 0 {
		portsJSON, err = json.Marshal(task.Ports)
		if err != nil {
			return fmt.Errorf("failed to marshal ports: %w", err)
		}
	}
	resourcesJSON, err = json.Marshal(task.Resources)
	if err != nil {
		return fmt.Errorf("failed to marshal resources: %w", err)
	}

	query := `
		INSERT INTO tasks (task_id, name, image, env, status, node_id, container_id, created_at, started_at, finished_at, error, liveness_probe, readiness_probe, restart_policy, health_status, ports, resources)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
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
		nullBytes(portsJSON),
		resourcesJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	return nil
}

func (s *PostgresStateStore) GetTask(taskID string) (types.Task, error) {
	query := `
		SELECT task_id, name, image, env, status, node_id, container_id, created_at, started_at, finished_at, error,
		       liveness_probe, readiness_probe, restart_policy, health_status, ports, resources
		FROM tasks
		WHERE task_id = $1
	`

	var task types.Task
	var envJSON, livenessProbeJSON, readinessProbeJSON, portsJSON, resourcesJSON []byte
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
		&portsJSON,
		&resourcesJSON,
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

	if len(portsJSON) > 0 {
		if err := json.Unmarshal(portsJSON, &task.Ports); err != nil {
			return types.Task{}, fmt.Errorf("failed to unmarshal ports: %w", err)
		}
	}

	if len(resourcesJSON) > 0 {
		if err := json.Unmarshal(resourcesJSON, &task.Resources); err != nil {
			return types.Task{}, fmt.Errorf("failed to unmarshal resources: %w", err)
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

func (s *PostgresStateStore) UpdateTask(taskID string, updates TaskUpdate) error {
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

func (s *PostgresStateStore) ListTasks() ([]types.Task, error) {
	query := `
		SELECT task_id, name, image, env, status, node_id, container_id, created_at, started_at, finished_at, error,
		       liveness_probe, readiness_probe, restart_policy, health_status, ports, resources
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
		var envJSON, livenessProbeJSON, readinessProbeJSON, portsJSON, resourcesJSON []byte
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
			&portsJSON,
			&resourcesJSON,
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

		if len(portsJSON) > 0 {
			if err := json.Unmarshal(portsJSON, &task.Ports); err != nil {
				return nil, fmt.Errorf("failed to unmarshal ports: %w", err)
			}
		}

		if len(resourcesJSON) > 0 {
			if err := json.Unmarshal(resourcesJSON, &task.Resources); err != nil {
				return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
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

func (s *PostgresStateStore) DeleteTask(taskID string) error {
	result, err := s.db.Exec("DELETE FROM tasks WHERE task_id = $1", taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTaskNotFound
	}

	return nil
}
