package models

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAutomation_ParseDefinition(t *testing.T) {
	t.Run("valid definition", func(t *testing.T) {
		expected := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{
				{
					Device: "sensor1", Action: "read_temp",
					Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25.5}},
				},
				{
					Device: "sensor2", Action: "read_temp",
					Conditions: []AutomationCondition{{Field: "value", Operator: "<", Threshold: 60}},
				},
			},
			Actions: []AutomationAction{
				{Device: "actuator1", Action: "turn_on"},
			},
		}
		data, _ := yaml.Marshal(expected)
		a := &Automation{Definition: string(data)}

		actual, err := a.ParseDefinition()
		require.NoError(t, err)
		assert.Equal(t, &expected, actual)
	})

	t.Run("invalid YAML", func(t *testing.T) {
		a := &Automation{Definition: "not: valid: yaml: [["}
		_, err := a.ParseDefinition()
		assert.Error(t, err)
	})

	t.Run("empty definition", func(t *testing.T) {
		a := &Automation{Definition: ""}
		actual, err := a.ParseDefinition()
		require.NoError(t, err)
		assert.Equal(t, &AutomationDefinition{}, actual)
	})
}

func TestAutomation_Validate(t *testing.T) {
	t.Run("invalid YAML returns error", func(t *testing.T) {
		a := Automation{Definition: "not: valid: yaml: [["}
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "invalid YAML definition")
	})

	t.Run("invalid condition_logic returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			ConditionLogic: "invalid",
			Actions:        []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "condition_logic must be 'and' or 'or'")
	})

	t.Run("invalid interval returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "invalid",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "interval must be a valid duration")
	})

	t.Run("invalid operator returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: "invalid", Threshold: 25}},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "invalid operator 'invalid'")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("trigger device not found returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}))

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "device 'sensor1' not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("trigger action not found returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "action 'read_temp' not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("trigger action not assigned to device returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[2,3]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "action 'read_temp' is not assigned to device 'sensor1'")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("trigger action not assigned when device has empty actions", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow(""))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "action 'read_temp' is not assigned to device 'sensor1'")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("action device not found returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("actuator1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}))

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "device 'actuator1' not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("action not found returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("actuator1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[2]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("turn_on").
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "action 'turn_on' not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("action not assigned to device returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("actuator1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[3,4]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("turn_on").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "action 'turn_on' is not assigned to device 'actuator1'")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("valid automation passes", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("actuator1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[2]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("turn_on").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

		err = a.Validate(context.Background(), db)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("multiple actions all valid passes", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{
				{Device: "actuator1", Action: "turn_on"},
				{Device: "actuator2", Action: "send_alert"},
			},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("actuator1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[2]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("turn_on").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("actuator2").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[3]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("send_alert").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))

		err = a.Validate(context.Background(), db)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("trigger with device but no action returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{Device: "sensor1"}},
			Actions:  []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "trigger must have both device and action")
	})

	t.Run("trigger with action but no device returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{Action: "read_temp"}},
			Actions:  []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "trigger must have both device and action")
	})

	t.Run("trigger with empty conditions returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "conditions are required when a trigger reads from a device")
	})

	t.Run("all valid operators pass", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device: "sensor1",
				Action: "read_temp",
				Conditions: []AutomationCondition{
					{Field: "v1", Operator: ">", Threshold: 1},
					{Field: "v2", Operator: "<", Threshold: 2},
					{Field: "v3", Operator: ">=", Threshold: 3},
					{Field: "v4", Operator: "<=", Threshold: 4},
					{Field: "v5", Operator: "==", Threshold: 5},
					{Field: "v6", Operator: "!=", Threshold: 6},
				},
			}},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("actuator1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[2]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("turn_on").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

		err = a.Validate(context.Background(), db)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty actions returns error", func(t *testing.T) {
		def := AutomationDefinition{
			Interval: "5m",
			Triggers: []AutomationTrigger{{
				Device:     "sensor1",
				Action:     "read_temp",
				Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
			}},
			Actions: []AutomationAction{},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err = a.Validate(context.Background(), db)
		assert.ErrorContains(t, err, "actions are required")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("multiple triggers with independent conditions", func(t *testing.T) {
		def := AutomationDefinition{
			Interval:       "5m",
			ConditionLogic: "and",
			Triggers: []AutomationTrigger{
				{
					Device:     "sensor1",
					Action:     "read_temp",
					Conditions: []AutomationCondition{{Field: "value", Operator: ">", Threshold: 25}},
				},
				{
					Device:     "sensor2",
					Action:     "read_humidity",
					Conditions: []AutomationCondition{{Field: "value", Operator: "<", Threshold: 60}},
				},
			},
			Actions: []AutomationAction{{Device: "actuator1", Action: "turn_on"}},
		}
		data, _ := yaml.Marshal(def)
		a := Automation{Definition: string(data)}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close() // nolint

		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[1]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_temp").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("sensor2").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[3]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("read_humidity").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))
		mock.ExpectQuery("SELECT actions FROM devices WHERE name = ?").
			WithArgs("actuator1").
			WillReturnRows(sqlmock.NewRows([]string{"actions"}).AddRow("[2]"))
		mock.ExpectQuery("SELECT id FROM actions WHERE name = ?").
			WithArgs("turn_on").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

		err = a.Validate(context.Background(), db)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
