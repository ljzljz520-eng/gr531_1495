package domain

type Column struct {
	Name       string `json:"name"`
	SourceType string `json:"sourceType"`
	Nullable   bool   `json:"nullable"`
}

type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Method  string   `json:"method"`
}

type Table struct {
	Name       string   `json:"name"`
	Columns    []Column `json:"columns"`
	PrimaryKey []string `json:"primaryKey"`
	Indexes    []Index  `json:"indexes"`
}

type Catalog struct {
	Tables []Table `json:"tables"`
}

type TypeChange struct {
	Column string `json:"column"`
	From   string `json:"from"`
	To     string `json:"to"`
}

type IndexChange struct {
	Index      string `json:"index"`
	FromMethod string `json:"fromMethod"`
	ToMethod   string `json:"toMethod"`
}

type TablePlan struct {
	Table        string        `json:"table"`
	Script       string        `json:"script"`
	TypeChanges  []TypeChange  `json:"typeChanges"`
	IndexChanges []IndexChange `json:"indexChanges"`
}

type Preview struct {
	Fixture string      `json:"fixture"`
	Plans   []TablePlan `json:"plans"`
}

type TableSummary struct {
	Name        string `json:"name"`
	ColumnCount int    `json:"columnCount"`
	IndexCount  int    `json:"indexCount"`
}

type TablePage struct {
	Items      []TableSummary `json:"items"`
	NextCursor string         `json:"nextCursor,omitempty"`
	Total      int            `json:"total"`
}

type ExecutionStatus string

const (
	ExecutionRunning   ExecutionStatus = "running"
	ExecutionCompleted ExecutionStatus = "completed"
	ExecutionFailed    ExecutionStatus = "failed"
)

func (s ExecutionStatus) Valid() bool {
	switch s {
	case ExecutionRunning, ExecutionCompleted, ExecutionFailed:
		return true
	default:
		return false
	}
}

type TableExecution struct {
	Table  string `json:"table"`
	Result string `json:"result"`
}

type Execution struct {
	ID     string           `json:"id"`
	Status ExecutionStatus  `json:"status"`
	Tables []TableExecution `json:"tables"`
}
