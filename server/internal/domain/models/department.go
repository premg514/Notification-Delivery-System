package models

import "strings"

type Department string

const (
	DepartmentCSE   Department = "CSE"
	DepartmentECE   Department = "ECE"
	DepartmentME    Department = "ME"
	DepartmentCivil Department = "CIVIL"
	DepartmentEEE   Department = "EEE"
)

func AllDepartments() []Department {
	return []Department{
		DepartmentCSE,
		DepartmentECE,
		DepartmentME,
		DepartmentCivil,
		DepartmentEEE,
	}
}

func ParseDepartment(value string) (Department, bool) {
	switch Department(strings.ToUpper(strings.TrimSpace(value))) {
	case DepartmentCSE:
		return DepartmentCSE, true
	case DepartmentECE:
		return DepartmentECE, true
	case DepartmentME:
		return DepartmentME, true
	case DepartmentCivil:
		return DepartmentCivil, true
	case DepartmentEEE:
		return DepartmentEEE, true
	default:
		return "", false
	}
}
