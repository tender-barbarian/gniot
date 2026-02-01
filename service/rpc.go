package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  any `json:"params,omitempty"`
	ID      int         `json:"id"`
}

func (s *Service[M, N]) callJSONRPC(ctx context.Context, ip, method, params string) error {
	var paramsObj interface{}
	if params != "" {
		if err := json.Unmarshal([]byte(params), &paramsObj); err != nil {
			return fmt.Errorf("parsing action params: %w", err)
		}
	}

	data := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsObj,
		ID:      1,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling JSON-RPC request: %w", err)
	}

	url := fmt.Sprintf("http://%s/rpc", ip)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("constructing HTTP call: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("calling device: %w", err)
	}
	defer resp.Body.Close() // nolint

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("device returned status %d", resp.StatusCode)
	}

	return nil
}
