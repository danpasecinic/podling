package state

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/danpasecinic/podling/internal/types"
)

func (s *PostgresStateStore) AddService(service types.Service) error {
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

func (s *PostgresStateStore) GetService(serviceID string) (types.Service, error) {
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

func (s *PostgresStateStore) GetServiceByName(namespace, name string) (types.Service, error) {
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

func (s *PostgresStateStore) UpdateService(serviceID string, updates types.ServiceUpdate) error {
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

func (s *PostgresStateStore) ListServices(namespace string) ([]types.Service, error) {
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

func (s *PostgresStateStore) DeleteService(serviceID string) error {
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

func (s *PostgresStateStore) SetEndpoints(endpoints types.Endpoints) error {
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

func (s *PostgresStateStore) GetEndpoints(serviceID string) (types.Endpoints, error) {
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

func (s *PostgresStateStore) GetEndpointsByServiceName(namespace, serviceName string) (types.Endpoints, error) {
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

func (s *PostgresStateStore) DeleteEndpoints(serviceID string) error {
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
