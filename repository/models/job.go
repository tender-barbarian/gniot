package models

import gocrud "github.com/tender-barbarian/go-crud"

type Job struct {
	ID       int    `json:"id" db:"id"`
	Name     string `json:"name" db:"name"`
	Devices  string `json:"devices,omitempty" db:"devices"`
	Action   string `json:"action,omitempty" db:"action"`
	RunAt    string `json:"run_at,omitempty" db:"run_at"`
	Interval string `json:"interval,omitempty" db:"interval"`
	gocrud.Reflection
}
