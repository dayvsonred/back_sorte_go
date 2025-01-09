package models

type Role struct {
	ID       int64  `json:"id" db:"id"`
	RoleName string `json:"role_name" db:"role_name"`
}
