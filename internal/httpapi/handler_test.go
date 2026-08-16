package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/order-schema-console/internal/domain"
	"example.com/order-schema-console/internal/httpapi"
	"example.com/order-schema-console/internal/service"
)

func TestTableQueryReturnsStablePage(t *testing.T) {
	recorder := request(t, service.New(), http.MethodGet, "/api/v1/source/tables?limit=2", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var page domain.TablePage
	decode(t, recorder, &page)
	if page.Total != 4 || page.NextCursor != "2" {
		t.Fatalf("page = %+v, want total 4 and next cursor 2", page)
	}
	if len(page.Items) != 2 || page.Items[0].Name != "orders" || page.Items[1].Name != "payments" {
		t.Fatalf("items = %+v, want orders followed by payments", page.Items)
	}
}

func TestPreviewShowsTypeAndIndexChanges(t *testing.T) {
	recorder := request(t, service.New(), http.MethodPost, "/api/v1/conversions/preview", `{"fixture":"default"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var preview domain.Preview
	decode(t, recorder, &preview)
	if len(preview.Plans) != 4 {
		t.Fatalf("plan count = %d, want 4", len(preview.Plans))
	}
	orders := plan(preview, "orders")
	if !hasTypeChange(orders, "placed_at", "DATETIME", "TIMESTAMPTZ") {
		t.Fatalf("orders type changes = %+v, want placed_at DATETIME to TIMESTAMPTZ", orders.TypeChanges)
	}
	payments := plan(preview, "payments")
	if len(payments.IndexChanges) != 1 || payments.IndexChanges[0].Index != "idx_payments_provider" || payments.IndexChanges[0].ToMethod != "BTREE" {
		t.Fatalf("payments index changes = %+v, want provider index converted to BTREE", payments.IndexChanges)
	}
	if !strings.Contains(orders.Script, `CREATE TABLE "orders"`) {
		t.Fatalf("orders script = %q, want CREATE TABLE statement", orders.Script)
	}
}

func TestExecutionSkipsTablesAlreadyInTarget(t *testing.T) {
	svc := service.New()
	recorder := request(t, svc, http.MethodPost, "/api/v1/executions", `{"fixture":"default","skipExisting":true}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	var execution domain.Execution
	decode(t, recorder, &execution)
	if execution.ID != "exec-000001" || execution.Status != domain.ExecutionCompleted {
		t.Fatalf("execution = %+v, want exec-000001 completed", execution)
	}
	if result(execution, "payments") != "skipped" || result(execution, "orders") != "created" {
		t.Fatalf("table results = %+v, want payments skipped and orders created", execution.Tables)
	}
	status := request(t, svc, http.MethodGet, "/api/v1/executions/exec-000001", "")
	if status.Code != http.StatusOK {
		t.Fatalf("status query = %d, want %d", status.Code, http.StatusOK)
	}
}

func TestPreviewIdentifiesUnsupportedOrderField(t *testing.T) {
	recorder := request(t, service.New(), http.MethodPost, "/api/v1/conversions/preview", `{"fixture":"unsupported-order-field"}`)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
	var response struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	decode(t, recorder, &response)
	if response.Code != "conversion_failed" || !strings.Contains(response.Message, "orders") || !strings.Contains(response.Message, "delivery_route") {
		t.Fatalf("conversion error = %+v, want the affected order field", response)
	}
}

func request(t *testing.T, svc *service.Service, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	httpapi.New(svc).Router().ServeHTTP(recorder, httptest.NewRequest(method, target, bytes.NewBufferString(body)))
	return recorder
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func plan(preview domain.Preview, table string) domain.TablePlan {
	for _, item := range preview.Plans {
		if item.Table == table {
			return item
		}
	}
	return domain.TablePlan{}
}

func hasTypeChange(plan domain.TablePlan, column, from, to string) bool {
	for _, change := range plan.TypeChanges {
		if change.Column == column && change.From == from && change.To == to {
			return true
		}
	}
	return false
}

func result(execution domain.Execution, table string) string {
	for _, item := range execution.Tables {
		if item.Table == table {
			return item.Result
		}
	}
	return ""
}
