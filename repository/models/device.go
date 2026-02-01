package models

import gocrud "github.com/tender-barbarian/go-crud"

type Device struct {
	ID      int    `json:"id" db:"id"`
	Name    string `json:"name" db:"name"`
	Type    string `json:"type" db:"type"`
	Chip    string `json:"chip" db:"chip"`
	Board   string `json:"board" db:"board"`
	IP      string `json:"ip" db:"ip"`
	Actions string `json:"actions,omitempty" db:"actions"`
	gocrud.Reflection
}

