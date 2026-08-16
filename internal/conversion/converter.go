package conversion

import (
	"fmt"
	"regexp"
	"strings"

	"example.com/order-schema-console/internal/domain"
)

var (
	varcharPattern = regexp.MustCompile(`^VARCHAR\(([1-9][0-9]*)\)$`)
	charPattern    = regexp.MustCompile(`^CHAR\(([1-9][0-9]*)\)$`)
	decimalPattern = regexp.MustCompile(`^DECIMAL\(([1-9][0-9]*),([0-9]+)\)$`)
)

type UnsupportedTypeError struct {
	SourceType string
}

func (e *UnsupportedTypeError) Error() string {
	return fmt.Sprintf("unsupported source type %q", e.SourceType)
}

type Converter struct{}

func New() Converter {
	return Converter{}
}

func (Converter) ConvertCatalog(catalog domain.Catalog) ([]domain.TablePlan, error) {
	plans := make([]domain.TablePlan, 0, len(catalog.Tables))
	for _, table := range catalog.Tables {
		plan, err := convertTable(table)
		if err != nil {
			return nil, fmt.Errorf("table %q: %w", table.Name, err)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func convertTable(table domain.Table) (domain.TablePlan, error) {
	definitions := make([]string, 0, len(table.Columns)+1)
	typeChanges := make([]domain.TypeChange, 0)
	for _, column := range table.Columns {
		targetType, err := targetType(column.SourceType)
		if err != nil {
			return domain.TablePlan{}, fmt.Errorf("column %q: %w", column.Name, err)
		}
		nullability := " NOT NULL"
		if column.Nullable {
			nullability = ""
		}
		definitions = append(definitions, fmt.Sprintf("  %s %s%s", quote(column.Name), targetType, nullability))
		if strings.ToUpper(column.SourceType) != targetType {
			typeChanges = append(typeChanges, domain.TypeChange{Column: column.Name, From: strings.ToUpper(column.SourceType), To: targetType})
		}
	}
	if len(table.PrimaryKey) > 0 {
		definitions = append(definitions, "  PRIMARY KEY ("+quoteList(table.PrimaryKey)+")")
	}

	var script strings.Builder
	fmt.Fprintf(&script, "CREATE TABLE %s (\n%s\n);", quote(table.Name), strings.Join(definitions, ",\n"))
	indexChanges := make([]domain.IndexChange, 0)
	for _, index := range table.Indexes {
		method := strings.ToUpper(index.Method)
		if method == "" {
			method = "BTREE"
		}
		targetMethod := method
		if method == "HASH" {
			targetMethod = "BTREE"
			indexChanges = append(indexChanges, domain.IndexChange{Index: index.Name, FromMethod: method, ToMethod: targetMethod})
		}
		unique := ""
		if index.Unique {
			unique = "UNIQUE "
		}
		fmt.Fprintf(&script, "\nCREATE %sINDEX %s ON %s USING %s (%s);", unique, quote(index.Name), quote(table.Name), targetMethod, quoteList(index.Columns))
	}
	return domain.TablePlan{
		Table:        table.Name,
		Script:       script.String(),
		TypeChanges:  typeChanges,
		IndexChanges: indexChanges,
	}, nil
}

func targetType(source string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(source))
	switch normalized {
	case "BIGINT", "TEXT":
		return normalized, nil
	case "INT":
		return "INTEGER", nil
	case "DATETIME":
		return "TIMESTAMPTZ", nil
	case "JSON":
		return "JSONB", nil
	case "BLOB":
		return "BYTEA", nil
	case "TINYINT(1)":
		return "BOOLEAN", nil
	}
	if varcharPattern.MatchString(normalized) || charPattern.MatchString(normalized) {
		return normalized, nil
	}
	if match := decimalPattern.FindStringSubmatch(normalized); match != nil {
		return fmt.Sprintf("NUMERIC(%s,%s)", match[1], match[2]), nil
	}
	return "", &UnsupportedTypeError{SourceType: normalized}
}

func quote(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func quoteList(identifiers []string) string {
	quoted := make([]string, len(identifiers))
	for i, identifier := range identifiers {
		quoted[i] = quote(identifier)
	}
	return strings.Join(quoted, ", ")
}
