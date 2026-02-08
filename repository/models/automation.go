package models

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	gocrud "github.com/tender-barbarian/go-crud"
	"gopkg.in/yaml.v3"
)

type Automation struct {
	ID              int             `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	Enabled         bool            `json:"enabled" db:"enabled"`
	Definition      string          `json:"definition" db:"definition"`
	LastCheck       string          `json:"last_check" db:"last_check"`
	LastTriggersRun string          `json:"last_triggers_run" db:"last_triggers_run"`
	LastActionRun   string          `json:"last_action_run" db:"last_action_run"`
	CreatedAt       gocrud.NullTime `json:"created_at" db:"created_at"`
	UpdatedAt       gocrud.NullTime `json:"updated_at" db:"updated_at"`
	gocrud.Reflection
}

type AutomationDefinition struct {
	Interval       string              `yaml:"interval"`
	Triggers       []AutomationTrigger `yaml:"triggers"`
	ConditionLogic string              `yaml:"condition_logic,omitempty"`
	Actions        []AutomationAction  `yaml:"actions"`
}

type AutomationTrigger struct {
	Device     string                `yaml:"device"`
	Action     string                `yaml:"action"`
	Conditions []AutomationCondition `yaml:"conditions"`
}

type AutomationCondition struct {
	Field     string  `yaml:"field"`
	Operator  string  `yaml:"operator"`
	Threshold float64 `yaml:"threshold"`
}

type AutomationAction struct {
	Device string `yaml:"device"`
	Action string `yaml:"action"`
}

func (a *Automation) ParseDefinition() (*AutomationDefinition, error) {
	var def AutomationDefinition
	if err := yaml.Unmarshal([]byte(a.Definition), &def); err != nil {
		return nil, err
	}
	return &def, nil
}

func (a *Automation) Validate(ctx context.Context, db gocrud.DBQuerier) error {
	def, err := a.ParseDefinition()
	if err != nil {
		return ValidationError{msg: "invalid YAML definition: " + err.Error()}
	}

	// Validate condition_logic is "and" or "or" (if provided)
	if def.ConditionLogic != "" && def.ConditionLogic != "and" && def.ConditionLogic != "or" {
		return ValidationError{msg: "condition_logic must be 'and' or 'or'"}
	}

	// Validate interval
	interval, err := time.ParseDuration(def.Interval)
	if err != nil {
		return ValidationError{msg: fmt.Errorf("interval must be a valid duration (e.g., '5m', '1h'): %w", err).Error()}
	}

	if interval < time.Second {
		return ValidationError{msg: "interval must be at least 1s"}
	}

	validOperators := map[string]bool{">": true, "<": true, ">=": true, "<=": true, "==": true, "!=": true}

	for _, trigger := range def.Triggers {
		// Triggers must have both device and action
		if trigger.Device == "" || trigger.Action == "" {
			return ValidationError{msg: "trigger must have both device and action"}
		}

		if err := validateDeviceAction(ctx, db, trigger.Device, trigger.Action); err != nil {
			return err
		}

		// Each trigger must have conditions to evaluate the response
		if len(trigger.Conditions) == 0 {
			return ValidationError{msg: "conditions are required when a trigger reads from a device"}
		}

		for _, cond := range trigger.Conditions {
			if cond.Field == "" {
				return ValidationError{msg: "condition must have a field"}
			}
			if !validOperators[cond.Operator] {
				return ValidationError{msg: fmt.Sprintf("invalid operator '%s': must be one of >, <, >=, <=, ==, !=", cond.Operator)}
			}
		}
	}

	if len(def.Actions) == 0 {
		return ValidationError{msg: "actions are required"}
	}

	// Validate all action devices and actions exist and are linked
	for _, action := range def.Actions {
		if err := validateDeviceAction(ctx, db, action.Device, action.Action); err != nil {
			return err
		}
	}

	return nil
}

func validateDeviceAction(ctx context.Context, db gocrud.DBQuerier, deviceName, actionName string) error {
	var deviceActions string
	row := db.QueryRowContext(ctx, "SELECT actions FROM devices WHERE name = ?", deviceName)
	if err := row.Scan(&deviceActions); err != nil {
		return ValidationError{msg: fmt.Sprintf("device '%s' not found", deviceName)}
	}

	var actionID int
	row = db.QueryRowContext(ctx, "SELECT id FROM actions WHERE name = ?", actionName)
	if err := row.Scan(&actionID); err != nil {
		return ValidationError{msg: fmt.Sprintf("action '%s' not found", actionName)}
	}

	var deviceActionIDs []int
	if deviceActions != "" {
		if err := json.Unmarshal([]byte(deviceActions), &deviceActionIDs); err != nil {
			return ValidationError{msg: fmt.Sprintf("failed to parse device '%s' actions", deviceName)}
		}
	}

	if !slices.Contains(deviceActionIDs, actionID) {
		return ValidationError{msg: fmt.Sprintf("action '%s' is not assigned to device '%s'", actionName, deviceName)}
	}

	return nil
}
