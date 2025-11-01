package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/danpasecinic/podling/internal/types"
)

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

	// Run migrations
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

// runMigrations applies database schema
func (s *PostgresStore) runMigrations() error {
	schema := `
		CREATE TABLE IF NOT EXISTS tasks (
			task_id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			image VARCHAR(255) NOT NULL,
			env JSONB,
			status VARCHAR(50) NOT NULL,
			node_id VARCHAR(255),
			container_id VARCHAR(255),
			created_at TIMESTAMP NOT NULL,
			started_at TIMESTAMP,
			finished_at TIMESTAMP,
			error TEXT
		);

		CREATE TABLE IF NOT EXISTS nodes (
			node_id VARCHAR(255) PRIMARY KEY,
			hostname VARCHAR(255) NOT NULL,
			port INTEGER NOT NULL,
			status VARCHAR(50) NOT NULL,
			capacity INTEGER NOT NULL,
			running_tasks INTEGER NOT NULL DEFAULT 0,
			last_heartbeat TIMESTAMP NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_node_id ON tasks(node_id);
		CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);
		CREATE INDEX IF NOT EXISTS idx_nodes_last_heartbeat ON nodes(last_heartbeat);
	`

	_, err := s.db.Exec(schema)
	return err
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

	query := `
		INSERT INTO tasks (task_id, name, image, env, status, node_id, container_id, created_at, started_at, finished_at, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
	)

	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	return nil
}

// GetTask retrieves a task by ID
func (s *PostgresStore) GetTask(taskID string) (types.Task, error) {
	query := `
		SELECT task_id, name, image, env, status, node_id, container_id, created_at, started_at, finished_at, error
		FROM tasks
		WHERE task_id = $1
	`

	var task types.Task
	var envJSON []byte
	var nodeID, containerID, errorMsg sql.NullString

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

	task.NodeID = nodeID.String
	task.ContainerID = containerID.String
	task.Error = errorMsg.String

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
		SELECT task_id, name, image, env, status, node_id, container_id, created_at, started_at, finished_at, error
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
		var envJSON []byte
		var nodeID, containerID, errorMsg sql.NullString

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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		// Unmarshal env
		if len(envJSON) > 0 {
			if err := json.Unmarshal(envJSON, &task.Env); err != nil {
				return nil, fmt.Errorf("failed to unmarshal env: %w", err)
			}
		}

		task.NodeID = nodeID.String
		task.ContainerID = containerID.String
		task.Error = errorMsg.String

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
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
