package model

type UserRoleType = string

const (
	UserRoleTypeRoot  UserRoleType = "root"
	UserRoleTypeAdmin UserRoleType = "admin"
	UserRoleTypeUser  UserRoleType = "user"
)
