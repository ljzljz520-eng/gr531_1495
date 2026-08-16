package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"example.com/order-schema-console/internal/service"
)

type Handler struct {
	service *service.Service
}

type fixtureRequest struct {
	Fixture string `json:"fixture"`
}

type executeRequest struct {
	Fixture      string `json:"fixture"`
	SkipExisting *bool  `json:"skipExisting"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New(svc *service.Service) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) Router() http.Handler {
	router := chi.NewRouter()
	router.Get("/healthz", h.health)
	router.Get("/api/v1/source/tables", h.listTables)
	router.Post("/api/v1/conversions/preview", h.preview)
	router.Post("/api/v1/executions", h.execute)
	router.Get("/api/v1/executions/{id}", h.execution)
	return router
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) listTables(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_page", "limit must be an integer")
			return
		}
		limit = value
	}
	page, err := h.service.ListTables(r.URL.Query().Get("query"), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_page", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	var request fixtureRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	preview, err := h.service.Preview(request.Fixture)
	if err != nil {
		var missing *service.FixtureNotFoundError
		if errors.As(err, &missing) {
			writeError(w, http.StatusBadRequest, "fixture_not_found", err.Error())
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "conversion_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *Handler) execute(w http.ResponseWriter, r *http.Request) {
	var request executeRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	skipExisting := true
	if request.SkipExisting != nil {
		skipExisting = *request.SkipExisting
	}
	execution, err := h.service.Execute(request.Fixture, skipExisting)
	if err != nil {
		var exists *service.TableExistsError
		if errors.As(err, &exists) {
			writeJSON(w, http.StatusConflict, struct {
				Error     errorResponse `json:"error"`
				Execution any           `json:"execution"`
			}{Error: errorResponse{Code: "table_exists", Message: err.Error()}, Execution: execution})
			return
		}
		var missing *service.FixtureNotFoundError
		if errors.As(err, &missing) {
			writeError(w, http.StatusBadRequest, "fixture_not_found", err.Error())
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "conversion_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, execution)
}

func (h *Handler) execution(w http.ResponseWriter, r *http.Request) {
	execution, err := h.service.Execution(chi.URLParam(r, "id"))
	if err != nil {
		var missing *service.ExecutionNotFoundError
		if errors.As(err, &missing) {
			writeError(w, http.StatusNotFound, "execution_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "invalid_execution", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, execution)
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain one JSON object")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Code: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
