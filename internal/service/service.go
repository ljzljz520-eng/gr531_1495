package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"example.com/order-schema-console/internal/conversion"
	"example.com/order-schema-console/internal/domain"
	"example.com/order-schema-console/internal/fixture"
)

type FixtureNotFoundError struct {
	Name string
}

func (e *FixtureNotFoundError) Error() string {
	return fmt.Sprintf("fixture %q not found", e.Name)
}

type ExecutionNotFoundError struct {
	ID string
}

func (e *ExecutionNotFoundError) Error() string {
	return fmt.Sprintf("execution %q not found", e.ID)
}

type TableExistsError struct {
	Table string
}

func (e *TableExistsError) Error() string {
	return fmt.Sprintf("target table %q already exists", e.Table)
}

type Service struct {
	converter  conversion.Converter
	catalogs   map[string]domain.Catalog
	mu         sync.RWMutex
	existing   map[string]struct{}
	executions map[string]domain.Execution
	nextID     int
}

func New() *Service {
	existing := make(map[string]struct{})
	for _, table := range fixture.ExistingTargetTables() {
		existing[table] = struct{}{}
	}
	return &Service{
		converter:  conversion.New(),
		catalogs:   fixture.Catalogs(),
		existing:   existing,
		executions: make(map[string]domain.Execution),
	}
}

func (s *Service) ListTables(query, cursor string, limit int) (domain.TablePage, error) {
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 200 {
		return domain.TablePage{}, fmt.Errorf("limit must be between 1 and 200")
	}
	offset := 0
	if cursor != "" {
		value, err := strconv.Atoi(cursor)
		if err != nil || value < 0 {
			return domain.TablePage{}, fmt.Errorf("cursor must be a non-negative integer")
		}
		offset = value
	}
	catalog := s.catalogs[fixture.DefaultCatalog]
	query = strings.ToLower(strings.TrimSpace(query))
	items := make([]domain.TableSummary, 0, len(catalog.Tables))
	for _, table := range catalog.Tables {
		if query != "" && !strings.Contains(strings.ToLower(table.Name), query) {
			continue
		}
		items = append(items, domain.TableSummary{Name: table.Name, ColumnCount: len(table.Columns), IndexCount: len(table.Indexes)})
	}
	if offset > len(items) {
		return domain.TablePage{}, fmt.Errorf("cursor is beyond the result set")
	}
	end := min(offset+limit, len(items))
	page := domain.TablePage{Items: append([]domain.TableSummary(nil), items[offset:end]...), Total: len(items)}
	if end < len(items) {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

func (s *Service) Preview(fixtureName string) (domain.Preview, error) {
	if fixtureName == "" {
		fixtureName = fixture.DefaultCatalog
	}
	catalog, ok := s.catalogs[fixtureName]
	if !ok {
		return domain.Preview{}, &FixtureNotFoundError{Name: fixtureName}
	}
	plans, err := s.converter.ConvertCatalog(catalog)
	if err != nil {
		return domain.Preview{}, errors.New("schema conversion failed")
	}
	return domain.Preview{Fixture: fixtureName, Plans: plans}, nil
}

func (s *Service) Execute(fixtureName string, skipExisting bool) (domain.Execution, error) {
	preview, err := s.Preview(fixtureName)
	if err != nil {
		return domain.Execution{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	id := fmt.Sprintf("exec-%06d", s.nextID)
	execution := domain.Execution{ID: id, Status: domain.ExecutionRunning, Tables: make([]domain.TableExecution, 0, len(preview.Plans))}
	for _, plan := range preview.Plans {
		if _, ok := s.existing[plan.Table]; ok {
			if skipExisting {
				execution.Tables = append(execution.Tables, domain.TableExecution{Table: plan.Table, Result: "skipped"})
				continue
			}
			execution.Status = domain.ExecutionFailed
			execution.Tables = append(execution.Tables, domain.TableExecution{Table: plan.Table, Result: "already_exists"})
			s.executions[id] = execution
			return execution, &TableExistsError{Table: plan.Table}
		}
		s.existing[plan.Table] = struct{}{}
		execution.Tables = append(execution.Tables, domain.TableExecution{Table: plan.Table, Result: "created"})
	}
	execution.Status = domain.ExecutionCompleted
	s.executions[id] = execution
	return execution, nil
}

func (s *Service) Execution(id string) (domain.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	execution, ok := s.executions[id]
	if !ok {
		return domain.Execution{}, &ExecutionNotFoundError{ID: id}
	}
	if !execution.Status.Valid() {
		return domain.Execution{}, fmt.Errorf("execution %q has invalid status %q", id, execution.Status)
	}
	execution.Tables = append([]domain.TableExecution(nil), execution.Tables...)
	return execution, nil
}
