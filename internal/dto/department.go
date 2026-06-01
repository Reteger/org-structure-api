package dto

import "time"

type CreateDeptReq struct {
	Name     string `json:"name"`
	ParentID *int   `json:"parent_id,omitempty"`
}

type UpdateDeptReq struct {
	Name     *string `json:"name,omitempty"`
	ParentID *int    `json:"parent_id,omitempty"`
}

type DeptResponse struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	ParentID  *int           `json:"parent_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	Employees []EmployeeRes  `json:"employees,omitempty"`
	Children  []DeptResponse `json:"children,omitempty"`
}
