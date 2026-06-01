package repository

import (
	"context"
	"gorm.io/gorm"
	"org-structure-api/internal/models"
)

type DepartmentRepo struct{ databaseConnection *gorm.DB }

func NewDepartmentRepo(databaseConnection *gorm.DB) *DepartmentRepo {
	return &DepartmentRepo{databaseConnection: databaseConnection}
}

func (repo *DepartmentRepo) GetByID(requestContext context.Context, departmentID int) (*models.Department, error) {
	var departmentRecord models.Department
	operationError := repo.databaseConnection.WithContext(requestContext).First(&departmentRecord, departmentID).Error
	if operationError != nil {
		return nil, operationError
	}
	return &departmentRecord, nil
}

func (repo *DepartmentRepo) Create(requestContext context.Context, departmentRecord *models.Department) error {
	return repo.databaseConnection.WithContext(requestContext).Create(departmentRecord).Error
}

func (repo *DepartmentRepo) GetTreeByCTE(rootDepartmentID int, maxDepth int) ([]models.Department, error) {
	var flatDepartmentList []models.Department
	recursiveQuery := `
	WITH RECURSIVE dept_tree AS (
		SELECT id, name, parent_id, created_at, updated_at, 0 AS level, ARRAY[id] AS path
		FROM departments WHERE id = ?
		UNION ALL
		SELECT d.id, d.name, d.parent_id, d.created_at, d.updated_at, dt.level + 1, dt.path || d.id
		FROM departments d
		INNER JOIN dept_tree dt ON d.parent_id = dt.id
		WHERE dt.level < ? AND NOT d.id = ANY(dt.path)
	)
	SELECT id, name, parent_id, created_at, updated_at FROM dept_tree ORDER BY level, name
	`
	operationError := repo.databaseConnection.Raw(recursiveQuery, rootDepartmentID, maxDepth).Scan(&flatDepartmentList).Error
	return flatDepartmentList, operationError
}

func (repo *DepartmentRepo) HasAncestor(potentialAncestorID, descendantID int) (bool, error) {
	recursiveQuery := `
	WITH RECURSIVE ancestors AS (
		SELECT parent_id FROM departments WHERE id = ?
		UNION ALL
		SELECT d.parent_id FROM departments d
		INNER JOIN ancestors a ON d.id = a.parent_id
		WHERE d.parent_id IS NOT NULL
	)
	SELECT COUNT(*) > 0 FROM ancestors WHERE parent_id = ?
	`
	var cycleDetected bool
	operationError := repo.databaseConnection.Raw(recursiveQuery, descendantID, potentialAncestorID).Scan(&cycleDetected).Error
	return cycleDetected, operationError
}

func (repo *DepartmentRepo) GetEmployeesByDepartmentIDs(requestContext context.Context, departmentIDList []int) ([]models.Employee, error) {
	if len(departmentIDList) == 0 {
		return nil, nil
	}
	var employeeList []models.Employee
	operationError := repo.databaseConnection.WithContext(requestContext).Where("department_id IN ?", departmentIDList).Find(&employeeList).Error
	return employeeList, operationError
}
