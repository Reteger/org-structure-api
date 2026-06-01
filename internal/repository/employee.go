package repository

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"org-structure-api/internal/models"
)

var ErrRecordNotFound = errors.New("record not found")

type EmployeeRepo struct{ databaseConnection *gorm.DB }

func NewEmployeeRepo(databaseConnection *gorm.DB) *EmployeeRepo {
	return &EmployeeRepo{databaseConnection: databaseConnection}
}

func (repo *EmployeeRepo) Create(requestContext context.Context, employeeRecord *models.Employee) error {
	return repo.databaseConnection.WithContext(requestContext).Create(employeeRecord).Error
}

func (repo *EmployeeRepo) GetByDepartmentIDs(requestContext context.Context, departmentIDList []int) ([]models.Employee, error) {
	if len(departmentIDList) == 0 {
		return nil, nil
	}
	var employeeList []models.Employee
	operationError := repo.databaseConnection.WithContext(requestContext).Where("department_id IN ?", departmentIDList).Find(&employeeList).Error
	return employeeList, operationError
}
