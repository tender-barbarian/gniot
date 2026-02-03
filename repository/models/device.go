package models

import (
	"encoding/json"

	gocrud "github.com/tender-barbarian/go-crud"
)

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

func (d *Device) Validate() error {
	if d.Actions != "" {
		var actions []int
		if err := json.Unmarshal([]byte(d.Actions), &actions); err != nil {
			return ValidationError{msg: "actions must be a list of action IDs"}
		}
	}

	return nil
}
