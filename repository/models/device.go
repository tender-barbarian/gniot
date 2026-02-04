package models

import (
	"context"
	"encoding/json"

	gocrud "github.com/tender-barbarian/go-crud"
)

type Device struct {
	ID        int             `json:"id" db:"id"`
	Name      string          `json:"name" db:"name"`
	Type      string          `json:"type" db:"type"`
	Chip      string          `json:"chip" db:"chip"`
	Board     string          `json:"board" db:"board"`
	IP        string          `json:"ip" db:"ip"`
	Actions   string          `json:"actions,omitempty" db:"actions"`
	CreatedAt gocrud.NullTime `json:"created_at" db:"created_at"`
	UpdatedAt gocrud.NullTime `json:"updated_at" db:"updated_at"`
	gocrud.Reflection
}

func (d *Device) Validate(ctx context.Context, db gocrud.DBQuerier) error {
	var actions []int
	if d.Actions != "" {
		if err := json.Unmarshal([]byte(d.Actions), &actions); err != nil {
			return ValidationError{msg: "actions must be a list of action IDs"}
		}
	}

	for _, actionID := range actions {
		var exists bool
		row := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM actions WHERE ID = ?)", actionID)
		if err := row.Scan(&exists); err != nil {
			return ValidationError{msg: err.Error()}
		}

		if !exists {
			return ValidationError{msg: "all actions must exist"}
		}
	}

	return nil
}
