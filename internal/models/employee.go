package models

import "time"

type Employee struct {
	ID           int        `gorm:"primaryKey;autoIncrement" json:"id"`
	DepartmentID int        `gorm:"not null;index" json:"department_id"`
	FullName     string     `gorm:"type:varchar(200);not null" json:"full_name"`
	Position     string     `gorm:"type:varchar(200);not null" json:"position"`
	HiredAt      *time.Time `json:"hired_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
