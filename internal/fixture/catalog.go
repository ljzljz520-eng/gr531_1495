package fixture

import "example.com/order-schema-console/internal/domain"

const (
	DefaultCatalog     = "default"
	UnsupportedCatalog = "unsupported-order-field"
)

func Catalogs() map[string]domain.Catalog {
	base := defaultCatalog()
	unsupported := cloneCatalog(base)
	unsupported.Tables[0].Columns = append(unsupported.Tables[0].Columns, domain.Column{
		Name:       "delivery_route",
		SourceType: "GEOMETRY",
		Nullable:   true,
	})
	return map[string]domain.Catalog{
		DefaultCatalog:     base,
		UnsupportedCatalog: unsupported,
	}
}

func ExistingTargetTables() []string {
	return []string{"payments"}
}

func defaultCatalog() domain.Catalog {
	return domain.Catalog{Tables: []domain.Table{
		{
			Name: "orders",
			Columns: []domain.Column{
				{Name: "order_id", SourceType: "BIGINT"},
				{Name: "customer_id", SourceType: "BIGINT"},
				{Name: "status", SourceType: "VARCHAR(32)"},
				{Name: "currency", SourceType: "CHAR(3)"},
				{Name: "total_amount", SourceType: "DECIMAL(18,2)"},
				{Name: "placed_at", SourceType: "DATETIME"},
				{Name: "attributes", SourceType: "JSON", Nullable: true},
			},
			PrimaryKey: []string{"order_id"},
			Indexes: []domain.Index{
				{Name: "idx_orders_customer_status", Columns: []string{"customer_id", "status"}, Method: "BTREE"},
				{Name: "idx_orders_placed_at", Columns: []string{"placed_at"}, Method: "BTREE"},
			},
		},
		{
			Name: "payments",
			Columns: []domain.Column{
				{Name: "payment_id", SourceType: "BIGINT"},
				{Name: "order_id", SourceType: "BIGINT"},
				{Name: "provider", SourceType: "VARCHAR(32)"},
				{Name: "amount", SourceType: "DECIMAL(18,2)"},
				{Name: "paid_at", SourceType: "DATETIME", Nullable: true},
				{Name: "gateway_response", SourceType: "JSON", Nullable: true},
			},
			PrimaryKey: []string{"payment_id"},
			Indexes: []domain.Index{
				{Name: "idx_payments_order", Columns: []string{"order_id"}, Method: "BTREE"},
				{Name: "idx_payments_provider", Columns: []string{"provider"}, Method: "HASH"},
			},
		},
		{
			Name: "after_sales",
			Columns: []domain.Column{
				{Name: "case_id", SourceType: "BIGINT"},
				{Name: "order_id", SourceType: "BIGINT"},
				{Name: "kind", SourceType: "VARCHAR(24)"},
				{Name: "reason", SourceType: "TEXT", Nullable: true},
				{Name: "refund_amount", SourceType: "DECIMAL(18,2)"},
				{Name: "opened_at", SourceType: "DATETIME"},
				{Name: "evidence", SourceType: "BLOB", Nullable: true},
			},
			PrimaryKey: []string{"case_id"},
			Indexes: []domain.Index{
				{Name: "idx_after_sales_order", Columns: []string{"order_id"}, Method: "BTREE"},
			},
		},
		{
			Name: "product_snapshots",
			Columns: []domain.Column{
				{Name: "snapshot_id", SourceType: "BIGINT"},
				{Name: "order_id", SourceType: "BIGINT"},
				{Name: "sku", SourceType: "VARCHAR(64)"},
				{Name: "title", SourceType: "VARCHAR(255)"},
				{Name: "unit_price", SourceType: "DECIMAL(18,2)"},
				{Name: "quantity", SourceType: "INT"},
				{Name: "snapshot", SourceType: "JSON"},
			},
			PrimaryKey: []string{"snapshot_id"},
			Indexes: []domain.Index{
				{Name: "uk_product_snapshots_order_sku", Columns: []string{"order_id", "sku"}, Unique: true, Method: "BTREE"},
			},
		},
	}}
}

func cloneCatalog(source domain.Catalog) domain.Catalog {
	tables := make([]domain.Table, len(source.Tables))
	for i, table := range source.Tables {
		tables[i] = table
		tables[i].Columns = append([]domain.Column(nil), table.Columns...)
		tables[i].PrimaryKey = append([]string(nil), table.PrimaryKey...)
		tables[i].Indexes = make([]domain.Index, len(table.Indexes))
		for j, index := range table.Indexes {
			tables[i].Indexes[j] = index
			tables[i].Indexes[j].Columns = append([]string(nil), index.Columns...)
		}
	}
	return domain.Catalog{Tables: tables}
}
