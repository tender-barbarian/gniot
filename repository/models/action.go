package models

import (
	"context"
	"encoding/json"

	gocrud "github.com/tender-barbarian/go-crud"
)

type Action struct {
	ID     int    `json:"id" db:"id"`
	Name   string `json:"name" db:"name"`
	Path   string `json:"path" db:"path"`
	Params string `json:"params" db:"params"`
	gocrud.Reflection
}

func (a *Action) Validate(ctx context.Context, db gocrud.DBQuerier) error {
	if a.Params != "" {
		if !json.Valid([]byte(a.Params)) {
			return ValidationError{msg: "params must be valid JSON"}
		}
	}

	return nil
}
