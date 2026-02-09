package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tender-barbarian/gniotek/repository/models"
)

func (s *Service) RunAutomations(ctx context.Context, interval time.Duration, errCh chan<- error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.processAutomations(ctx); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}
	}
}

func (s *Service) processAutomations(ctx context.Context) error {
	now := time.Now()
	automations, err := s.automationsRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("getting automations: %w", err)
	}

	var hadErrors bool
	for _, automation := range automations {
		if !automation.Enabled {
			continue
		}

		s.logger.Info("processing automation", "automation", automation.Name)

		if err := s.processOneAutomation(ctx, automation, now); err != nil {
			s.logger.Error("automation failed", "automation", automation.Name, "error", err)
			hadErrors = true
		}
	}

	if hadErrors {
		return fmt.Errorf("one or more automations encountered errors")
	}

	return nil
}

func (s *Service) processOneAutomation(ctx context.Context, automation *models.Automation, now time.Time) error {
	automation.LastCheck = now.Format(time.RFC3339)
	if err := s.automationsRepo.Update(ctx, automation, automation.ID); err != nil {
		s.logger.Warn("failed to update last check", "automation", automation.Name, "error", err)
	}

	definition, err := automation.ParseDefinition()
	if err != nil {
		return fmt.Errorf("parsing definition: %w", err)
	}

	// Check if the automation interval has elapsed
	var lastTriggered time.Time
	if automation.LastTriggersRun != "" {
		lastTriggered, err = time.Parse(time.RFC3339, automation.LastTriggersRun)
		if err != nil {
			return fmt.Errorf("parsing last triggers run time: %w", err)
		}
	} else {
		lastTriggered = automation.CreatedAt.Time
	}

	interval, err := time.ParseDuration(definition.Interval)
	if err != nil {
		return fmt.Errorf("parsing interval: %w", err)
	}

	if lastTriggered.Add(interval).After(now) {
		return nil
	}

	results, err := s.processTriggers(ctx, definition)
	if err != nil {
		return fmt.Errorf("processing triggers: %w", err)
	}

	automation.LastTriggersRun = now.Format(time.RFC3339)
	if err := s.automationsRepo.Update(ctx, automation, automation.ID); err != nil {
		return fmt.Errorf("update triggers last run time: %w", err)
	}

	if !s.applyConditionLogic(results, definition.ConditionLogic) {
		return nil
	}

	for _, action := range definition.Actions {
		result, err := s.executeAction(ctx, action.Device, action.Action)
		if err != nil {
			return fmt.Errorf("executing action [%s] on device [%s]: %w", action.Action, action.Device, err)
		}

		automation.LastActionRun = now.Format(time.RFC3339)
		if err := s.automationsRepo.Update(ctx, automation, automation.ID); err != nil {
			return fmt.Errorf("update automation action last run time: %w", err)
		}

		s.logger.Info("successfully executed automation action", "automation", automation.Name, "action", action.Action, "device", action.Device, "response from device", result)
	}

	s.logger.Info("automation processed", "automation", automation.Name)
	return nil
}

func (s *Service) processTriggers(ctx context.Context, def *models.AutomationDefinition) ([]bool, error) {
	var results []bool
	for _, trigger := range def.Triggers {
		response, err := s.executeAction(ctx, trigger.Device, trigger.Action)
		if err != nil {
			return nil, fmt.Errorf("executing trigger, device [%s], action [%s]: %w", trigger.Device, trigger.Action, err)
		}

		s.logger.Info("successfully executed trigger", "device", trigger.Device, "action", trigger.Action, "response", response)

		met, err := s.evaluateConditions(response, trigger)
		if err != nil {
			return nil, fmt.Errorf("evaluating conditions for trigger [%s/%s]: %w", trigger.Device, trigger.Action, err)
		}
		results = append(results, met)
	}

	return results, nil
}

func (s *Service) executeAction(ctx context.Context, deviceName, actionName string) (map[string]any, error) {
	deviceID, err := s.devicesCache.GetIDByName(ctx, s.queryRepo, "devices", deviceName)
	if err != nil {
		return nil, fmt.Errorf("looking up device: %w", err)
	}

	actionID, err := s.actionsCache.GetIDByName(ctx, s.queryRepo, "actions", actionName)
	if err != nil {
		return nil, fmt.Errorf("looking up action: %w", err)
	}

	response, err := s.Execute(ctx, deviceID, actionID)
	if err != nil {
		return nil, fmt.Errorf("executing action [%s]: %w", actionName, err)
	}

	var parsedResponse map[string]any
	err = json.Unmarshal([]byte(response.Result), &parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("parsing trigger response, device [%s], action [%s]: %w", deviceName, actionName, err)
	}

	return parsedResponse, nil
}

func (s *Service) evaluateConditions(response map[string]any, trigger models.AutomationTrigger) (bool, error) {
	for _, condition := range trigger.Conditions {
		val, err := s.getFieldValue(response, condition.Field)
		if err != nil {
			return false, fmt.Errorf("getting field [%s] value: %w", condition.Field, err)
		}

		if !evaluateOperator(val, condition.Operator, condition.Threshold) {
			return false, nil
		}
	}

	return true, nil
}

func (s *Service) applyConditionLogic(results []bool, logic string) bool {
	if len(results) == 0 {
		return true
	}

	if logic == "or" {
		for _, r := range results {
			if r {
				return true
			}
		}
		return false
	}

	for _, r := range results {
		if !r {
			return false
		}
	}
	return true
}

func evaluateOperator(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	}

	return false
}

func (s *Service) getFieldValue(data map[string]any, field string) (float64, error) {
	parts := strings.Split(field, ".")

	var current any = data
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return 0, fmt.Errorf("field '%s' is not an object", part)
		}
		current, ok = m[part]
		if !ok {
			return 0, fmt.Errorf("field '%s' not found", part)
		}
	}

	switch v := current.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("field '%s' is not a number", field)
	}
}
