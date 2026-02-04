package models

import (
	"time"

	gocrud "github.com/tender-barbarian/go-crud"
)

type Job struct {
	ID            int       `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Devices       string    `json:"devices,omitempty" db:"devices"`
	Action        string    `json:"action,omitempty" db:"action"`
	RunAt         string    `json:"run_at,omitempty" db:"run_at"`
	Interval      string    `json:"interval,omitempty" db:"interval"`
	Enabled       int       `json:"enabled" db:"enabled"`
	LastCheck     string    `json:"last_check,omitempty" db:"last_check"`
	LastTriggered string    `json:"last_triggered,omitempty" db:"last_triggered"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	gocrud.Reflection
}
