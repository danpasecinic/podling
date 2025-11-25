package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/danpasecinic/podling/internal/types"
)

func (s *PostgresStateStore) AddPod(pod types.Pod) error {
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

func (s *PostgresStateStore) GetPod(podID string) (types.Pod, error) {
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

func (s *PostgresStateStore) UpdatePod(podID string, updates PodUpdate) error {
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

func (s *PostgresStateStore) ListPods() ([]types.Pod, error) {
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

func (s *PostgresStateStore) DeletePod(podID string) error {
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

func (s *PostgresStateStore) ListPodsByLabels(namespace string, labels map[string]string) ([]types.Pod, error) {
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
