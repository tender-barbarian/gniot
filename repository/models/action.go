package models

import gocrud "github.com/tender-barbarian/go-crud"

type Action struct {
	ID     int    `json:"id" db:"id"`
	Name   string `json:"name" db:"name"`
	Path   string `json:"path" db:"path"`
	Params string `json:"params" db:"params"`
	gocrud.Reflection
}
