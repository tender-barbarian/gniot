package models

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAction_Validate(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr string
	}{
		{
			name:    "empty params is valid",
			action:  Action{Params: ""},
			wantErr: "",
		},
		{
			name:    "valid JSON object",
			action:  Action{Params: `{"pin": 5}`},
			wantErr: "",
		},
		{
			name:    "valid JSON array",
			action:  Action{Params: `[1, 2, 3]`},
			wantErr: "",
		},
		{
			name:    "valid empty JSON object",
			action:  Action{Params: `{}`},
			wantErr: "",
		},
		{
			name:    "invalid JSON returns error",
			action:  Action{Params: "not-json"},
			wantErr: "params must be valid JSON",
		},
		{
			name:    "malformed JSON returns error",
			action:  Action{Params: `{"pin": }`},
			wantErr: "params must be valid JSON",
		},
		{
			name:    "unclosed brace returns error",
			action:  Action{Params: `{"pin": 5`},
			wantErr: "params must be valid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate(context.Background(), nil)

			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
