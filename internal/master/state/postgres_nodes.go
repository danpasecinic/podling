package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/danpasecinic/podling/internal/types"
)

func (s *PostgresStateStore) AddNode(node types.Node) error {
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

func (s *PostgresStateStore) GetNode(nodeID string) (types.Node, error) {
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

func (s *PostgresStateStore) UpdateNode(nodeID string, updates NodeUpdate) error {
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

func (s *PostgresStateStore) ListNodes() ([]types.Node, error) {
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

func (s *PostgresStateStore) DeleteNode(nodeID string) error {
	result, err := s.db.Exec("DELETE FROM nodes WHERE node_id = $1", nodeID)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNodeNotFound
	}

	return nil
}

func (s *PostgresStateStore) GetAvailableNodes() ([]types.Node, error) {
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
