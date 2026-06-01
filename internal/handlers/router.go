package handlers

import (
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"net/http"
	"org-structure-api/internal/config"
	"org-structure-api/internal/dto"
	"org-structure-api/internal/services"
	"strconv"
)

func New(deptSvc *services.DepartmentService, empSvc *services.EmployeeService, db *gorm.DB) *Handler {
	return &Handler{DeptSvc: deptSvc, EmpSvc: empSvc, DB: db}
}

type Handler struct {
	DeptSvc *services.DepartmentService
	EmpSvc  *services.EmployeeService
	DB      *gorm.DB
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /departments/", h.CreateDept)
	mux.HandleFunc("GET /departments/{id}", h.GetDept)
	mux.HandleFunc("PATCH /departments/{id}", h.UpdateDept)
	mux.HandleFunc("DELETE /departments/{id}", h.DeleteDept)
	mux.HandleFunc("POST /departments/{id}/employees/", h.CreateEmp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *Handler) CreateDept(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateDeptReq
	if decodeError := json.NewDecoder(r.Body).Decode(&request); decodeError != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	responseRecord, operationError := h.DeptSvc.Create(r.Context(), request)
	if operationError != nil {
		h.mapErr(w, operationError)
		return
	}
	writeJSON(w, http.StatusCreated, responseRecord)
}

func (h *Handler) GetDept(w http.ResponseWriter, r *http.Request) {
	identifierValue := r.PathValue("id")
	if identifierValue == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}
	departmentID, parseError := strconv.Atoi(identifierValue)
	if parseError != nil || departmentID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	depthValue := r.URL.Query().Get("depth")
	parsedDepth := config.DefaultTreeDepth
	if depthValue != "" {
		convertedDepth, conversionError := strconv.Atoi(depthValue)
		if conversionError == nil && convertedDepth >= config.MinTreeDepth && convertedDepth <= config.MaxTreeDepth {
			parsedDepth = convertedDepth
		}
	}

	includeValue := r.URL.Query().Get("include_employees")
	parsedInclude := config.DefaultIncludeEmployees
	if includeValue != "" {
		convertedInclude, conversionError := strconv.ParseBool(includeValue)
		if conversionError == nil {
			parsedInclude = convertedInclude
		}
	}

	responseRecord, operationError := h.DeptSvc.GetWithTree(r.Context(), departmentID, parsedDepth, parsedInclude)
	if operationError != nil {
		h.mapErr(w, operationError)
		return
	}
	writeJSON(w, http.StatusOK, responseRecord)
}

func (h *Handler) UpdateDept(w http.ResponseWriter, r *http.Request) {
	identifierValue := r.PathValue("id")
	if identifierValue == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}
	departmentID, parseError := strconv.Atoi(identifierValue)
	if parseError != nil || departmentID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var request dto.UpdateDeptReq
	if decodeError := json.NewDecoder(r.Body).Decode(&request); decodeError != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	var responseRecord *dto.DeptResponse
	transactionError := h.DB.Transaction(func(tx *gorm.DB) error {
		var operationError error
		responseRecord, operationError = h.DeptSvc.Update(r.Context(), tx, departmentID, request)
		return operationError
	})
	if transactionError != nil {
		h.mapErr(w, transactionError)
		return
	}
	writeJSON(w, http.StatusOK, responseRecord)
}

func (h *Handler) DeleteDept(w http.ResponseWriter, r *http.Request) {
	identifierValue := r.PathValue("id")
	if identifierValue == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}
	departmentID, parseError := strconv.Atoi(identifierValue)
	if parseError != nil || departmentID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	modeValue := r.URL.Query().Get("mode")
	var reassignTargetID *int
	reassignValue := r.URL.Query().Get("reassign_to_department_id")
	if reassignValue != "" {
		convertedID, conversionError := strconv.Atoi(reassignValue)
		if conversionError == nil {
			reassignTargetID = &convertedID
		}
	}

	transactionError := h.DB.Transaction(func(tx *gorm.DB) error {
		return h.DeptSvc.Delete(r.Context(), tx, departmentID, modeValue, reassignTargetID)
	})
	if transactionError != nil {
		h.mapErr(w, transactionError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateEmp(w http.ResponseWriter, r *http.Request) {
	identifierValue := r.PathValue("id")
	if identifierValue == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}
	departmentID, parseError := strconv.Atoi(identifierValue)
	if parseError != nil || departmentID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var request dto.CreateEmpReq
	if decodeError := json.NewDecoder(r.Body).Decode(&request); decodeError != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	responseRecord, operationError := h.EmpSvc.Create(r.Context(), departmentID, request)
	if operationError != nil {
		h.mapErr(w, operationError)
		return
	}
	writeJSON(w, http.StatusCreated, responseRecord)
}

func (h *Handler) mapErr(w http.ResponseWriter, operationError error) {
	switch {
	case errors.Is(operationError, services.ErrNotFound):
		writeError(w, http.StatusNotFound, operationError.Error())
	case errors.Is(operationError, services.ErrConflict):
		writeError(w, http.StatusConflict, operationError.Error())
	case errors.Is(operationError, services.ErrValidation):
		writeError(w, http.StatusBadRequest, operationError.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
