package models

type UserRole struct {
	IDUser string `json:"id_user" db:"id_user"`
	IDRole int64  `json:"id_role" db:"id_role"`
}
