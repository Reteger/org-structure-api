package services

import (
	"context"
	"org-structure-api/internal/config"
	"org-structure-api/internal/dto"
	"org-structure-api/internal/models"
	"org-structure-api/internal/repository"
	"strings"
)

type EmployeeService struct {
	repository     *repository.EmployeeRepo
	departmentRepo *repository.DepartmentRepo
}

func NewEmployeeService(employeeRepository *repository.EmployeeRepo, departmentRepository *repository.DepartmentRepo) *EmployeeService {
	return &EmployeeService{repository: employeeRepository, departmentRepo: departmentRepository}
}

func (svc *EmployeeService) Create(requestContext context.Context, departmentID int, request dto.CreateEmpReq) (*dto.EmployeeRes, error) {
	departmentRecord, fetchError := svc.departmentRepo.GetByID(requestContext, departmentID)
	if fetchError != nil || departmentRecord == nil {
		return nil, ErrNotFound
	}

	request.FullName = strings.TrimSpace(request.FullName)
	request.Position = strings.TrimSpace(request.Position)

	fullNameLength := len(request.FullName)
	positionLength := len(request.Position)

	if fullNameLength < config.MinNameLength || fullNameLength > config.MaxNameLength ||
		positionLength < config.MinNameLength || positionLength > config.MaxNameLength {
		return nil, ErrValidation
	}

	employeeRecord := &models.Employee{
		DepartmentID: departmentID,
		FullName:     request.FullName,
		Position:     request.Position,
		HiredAt:      request.HiredAt,
	}

	if operationError := svc.repository.Create(requestContext, employeeRecord); operationError != nil {
		return nil, operationError
	}

	return &dto.EmployeeRes{
		ID: employeeRecord.ID, DepartmentID: employeeRecord.DepartmentID, FullName: employeeRecord.FullName,
		Position: employeeRecord.Position, HiredAt: employeeRecord.HiredAt, CreatedAt: employeeRecord.CreatedAt,
	}, nil
}
