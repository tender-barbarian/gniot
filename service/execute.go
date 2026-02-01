package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"

	"github.com/tender-barbarian/gniot/repository/models"
)

func (s *Service[M, N, O]) Execute(ctx context.Context, deviceId, actionId int) error {
	d, err := s.devicesRepo.Get(ctx, deviceId)
	if err != nil {
		return fmt.Errorf("getting device: %w", err)
	}

	device, ok := any(d).(*models.Device)
	if !ok {
		return errors.New("asserting device type")
	}

	a, err := s.actionsRepo.Get(ctx, actionId)
	if err != nil {
		return fmt.Errorf("getting action: %w", err)
	}

	action, ok := any(a).(*models.Action)
	if !ok {
		return errors.New("asserting action type")
	}

	var actionIds []int
	err = json.Unmarshal([]byte(device.Actions), &actionIds)
	if err != nil {
		return fmt.Errorf("unmarshalling device actions: %w", err)
	}

	if !slices.Contains(actionIds, actionId) {
		return fmt.Errorf("action %d does not belong to device %d", actionId, deviceId)
	}

	if !isPrivateIP(device.IP) {
		return errors.New("device IP must be in private range")
	}

	return s.callJSONRPC(ctx, device.IP, action.Path, action.Params)
}

func isPrivateIP(ipStr string) bool {
	host, _, err := net.SplitHostPort(ipStr)
	if err != nil {
		host = ipStr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	if ip.IsPrivate() || ip.IsLoopback() {
		return true
	}

	return false
}
