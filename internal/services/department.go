package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"org-structure-api/internal/config"
	"org-structure-api/internal/dto"
	"org-structure-api/internal/models"
	"org-structure-api/internal/repository"
)

var (
	ErrNotFound   = errors.New("department not found")
	ErrConflict   = errors.New("structure conflict")
	ErrValidation = errors.New("validation failed")
)

type DepartmentService struct{ repository *repository.DepartmentRepo }

func NewDepartmentService(departmentRepository *repository.DepartmentRepo) *DepartmentService {
	return &DepartmentService{repository: departmentRepository}
}

func (svc *DepartmentService) Create(requestContext context.Context, request dto.CreateDeptReq) (*dto.DeptResponse, error) {
	request.Name = strings.TrimSpace(request.Name)
	nameLength := len(request.Name)

	if nameLength < config.MinNameLength || nameLength > config.MaxNameLength {
		return nil, ErrValidation
	}

	if request.ParentID == nil {
		return svc.createWithParent(requestContext, request, nil)
	}

	parentDepartment, fetchError := svc.repository.GetByID(requestContext, *request.ParentID)
	if fetchError != nil || parentDepartment == nil {
		return nil, ErrNotFound
	}
	return svc.createWithParent(requestContext, request, request.ParentID)
}

func (svc *DepartmentService) createWithParent(requestContext context.Context, request dto.CreateDeptReq, parentIdentifier *int) (*dto.DeptResponse, error) {
	departmentRecord := &models.Department{Name: request.Name, ParentID: parentIdentifier}
	operationError := svc.repository.Create(requestContext, departmentRecord)
	if operationError != nil {
		if isUniqueViolation(operationError) {
			return nil, fmt.Errorf("name conflict: %w", ErrConflict)
		}
		return nil, operationError
	}
	return toSimpleDeptResponse(departmentRecord), nil
}

func (svc *DepartmentService) GetWithTree(requestContext context.Context, departmentID int, depth int, includeEmployees bool) (*dto.DeptResponse, error) {
	if depth < config.MinTreeDepth || depth > config.MaxTreeDepth {
		depth = config.DefaultTreeDepth
	}

	rootDepartment, fetchError := svc.repository.GetByID(requestContext, departmentID)
	if fetchError != nil || rootDepartment == nil {
		return nil, ErrNotFound
	}

	flatDepartmentList, fetchError := svc.repository.GetTreeByCTE(departmentID, depth)
	if fetchError != nil {
		return nil, fmt.Errorf("get tree by CTE: %w", fetchError)
	}
	if len(flatDepartmentList) == 0 {
		return nil, ErrNotFound
	}

	treeRoot := buildTreeFromFlat(flatDepartmentList, departmentID)
	if treeRoot == nil {
		return nil, ErrNotFound
	}

	if !includeEmployees {
		return toDeptResponse(treeRoot), nil
	}

	departmentIDList := collectDepartmentIDs(treeRoot)
	if len(departmentIDList) == 0 {
		return toDeptResponse(treeRoot), nil
	}

	employeeList, fetchError := svc.repository.GetEmployeesByDepartmentIDs(requestContext, departmentIDList)
	if fetchError != nil {
		return nil, fmt.Errorf("get employees batch: %w", fetchError)
	}

	assignEmployeesToDepartments(treeRoot, employeeList)
	return toDeptResponse(treeRoot), nil
}

func buildTreeFromFlat(flatDepartmentList []models.Department, rootDepartmentID int) *models.Department {
	departmentMap := make(map[int]*models.Department, len(flatDepartmentList))
	for recordIndex := range flatDepartmentList {
		departmentItem := flatDepartmentList[recordIndex]
		departmentMap[departmentItem.ID] = &departmentItem
	}

	rootDepartment, isFound := departmentMap[rootDepartmentID]
	if !isFound {
		return nil
	}

	for recordIndex := range flatDepartmentList {
		departmentItem := flatDepartmentList[recordIndex]
		if departmentItem.ID == rootDepartmentID {
			continue
		}
		if departmentItem.ParentID == nil {
			continue
		}
		parentDepartment, isFound := departmentMap[*departmentItem.ParentID]
		if !isFound {
			continue
		}
		parentDepartment.Children = append(parentDepartment.Children, departmentItem)
	}

	return rootDepartment
}

func collectDepartmentIDs(rootDepartment *models.Department) []int {
	if rootDepartment == nil {
		return nil
	}
	var departmentIDList []int
	var walkFunc func(*models.Department)
	walkFunc = func(departmentNode *models.Department) {
		if departmentNode == nil {
			return
		}
		departmentIDList = append(departmentIDList, departmentNode.ID)
		for childIndex := range departmentNode.Children {
			walkFunc(&departmentNode.Children[childIndex])
		}
	}
	walkFunc(rootDepartment)
	return departmentIDList
}

func assignEmployeesToDepartments(rootDepartment *models.Department, employeeList []models.Employee) {
	if rootDepartment == nil {
		return
	}
	employeeMap := make(map[int][]models.Employee)
	for employeeIndex := range employeeList {
		employeeItem := employeeList[employeeIndex]
		employeeMap[employeeItem.DepartmentID] = append(employeeMap[employeeItem.DepartmentID], employeeItem)
	}
	var walkFunc func(*models.Department)
	walkFunc = func(departmentNode *models.Department) {
		if departmentNode == nil {
			return
		}
		departmentNode.Employees = employeeMap[departmentNode.ID]
		for childIndex := range departmentNode.Children {
			walkFunc(&departmentNode.Children[childIndex])
		}
	}
	walkFunc(rootDepartment)
}

func toDeptResponse(departmentNode *models.Department) *dto.DeptResponse {
	if departmentNode == nil {
		return nil
	}
	responseRecord := &dto.DeptResponse{
		ID:        departmentNode.ID,
		Name:      departmentNode.Name,
		ParentID:  departmentNode.ParentID,
		CreatedAt: departmentNode.CreatedAt,
	}
	for childIndex := range departmentNode.Children {
		responseRecord.Children = append(responseRecord.Children, *toDeptResponse(&departmentNode.Children[childIndex]))
	}
	for employeeIndex := range departmentNode.Employees {
		responseRecord.Employees = append(responseRecord.Employees, toEmpResponse(departmentNode.Employees[employeeIndex]))
	}
	return responseRecord
}

func (svc *DepartmentService) Update(requestContext context.Context, transaction *gorm.DB, departmentID int, request dto.UpdateDeptReq) (*dto.DeptResponse, error) {
	departmentRecord, fetchError := svc.repository.GetByID(requestContext, departmentID)
	if fetchError != nil || departmentRecord == nil {
		return nil, ErrNotFound
	}

	if validateError := svc.validateParentChange(departmentRecord.ID, request); validateError != nil {
		return nil, validateError
	}

	svc.applyChanges(departmentRecord, request)

	if saveError := transaction.WithContext(requestContext).Save(departmentRecord).Error; saveError != nil {
		return nil, svc.mapDatabaseError(saveError)
	}

	return toSimpleDeptResponse(departmentRecord), nil
}

func (svc *DepartmentService) validateParentChange(departmentID int, request dto.UpdateDeptReq) error {
	if request.ParentID == nil {
		return nil
	}
	newParentID := *request.ParentID
	if newParentID == departmentID {
		return fmt.Errorf("self-parent: %w", ErrConflict)
	}
	isCycle, checkError := svc.repository.HasAncestor(departmentID, newParentID)
	if checkError != nil {
		return fmt.Errorf("check cycle: %w", checkError)
	}
	if isCycle {
		return fmt.Errorf("cycle detected: %w", ErrConflict)
	}
	return nil
}

func (svc *DepartmentService) applyChanges(departmentRecord *models.Department, request dto.UpdateDeptReq) {
	if request.Name != nil {
		departmentRecord.Name = strings.TrimSpace(*request.Name)
	}
	if request.ParentID != nil {
		departmentRecord.ParentID = request.ParentID
	}
}

func (svc *DepartmentService) Delete(requestContext context.Context, transaction *gorm.DB, departmentID int, mode string, reassignTargetID *int) error {
	departmentRecord, fetchError := svc.repository.GetByID(requestContext, departmentID)
	if fetchError != nil || departmentRecord == nil {
		return ErrNotFound
	}

	if mode == config.DeleteModeCascade {
		return transaction.WithContext(requestContext).Delete(departmentRecord).Error
	}

	if mode != config.DeleteModeReassign {
		return ErrValidation
	}

	if reassignTargetID == nil {
		return ErrValidation
	}

	targetDepartment, fetchError := svc.repository.GetByID(requestContext, *reassignTargetID)
	if fetchError != nil || targetDepartment == nil {
		return ErrNotFound
	}

	if updateError := transaction.WithContext(requestContext).Model(&models.Employee{}).
		Where("department_id = ?", departmentID).
		Update("department_id", *reassignTargetID).Error; updateError != nil {
		return updateError
	}

	return transaction.WithContext(requestContext).Delete(departmentRecord).Error
}

func (svc *DepartmentService) mapDatabaseError(operationError error) error {
	if isUniqueViolation(operationError) {
		return fmt.Errorf("name conflict: %w", ErrConflict)
	}
	return fmt.Errorf("database operation failed: %w", operationError)
}

func isUniqueViolation(operationError error) bool {
	var databaseError *pgconn.PgError
	if errors.As(operationError, &databaseError) && databaseError.Code == config.PgErrUniqueViolation {
		return true
	}
	return false
}

func toSimpleDeptResponse(departmentNode *models.Department) *dto.DeptResponse {
	if departmentNode == nil {
		return nil
	}
	return &dto.DeptResponse{ID: departmentNode.ID, Name: departmentNode.Name, ParentID: departmentNode.ParentID, CreatedAt: departmentNode.CreatedAt}
}

func toEmpResponse(employeeRecord models.Employee) dto.EmployeeRes {
	return dto.EmployeeRes{
		ID: employeeRecord.ID, DepartmentID: employeeRecord.DepartmentID, FullName: employeeRecord.FullName,
		Position: employeeRecord.Position, HiredAt: employeeRecord.HiredAt, CreatedAt: employeeRecord.CreatedAt,
	}
}
