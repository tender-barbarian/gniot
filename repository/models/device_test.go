package models

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevice_ValidateWithDB(t *testing.T) {
	tests := []struct {
		name      string
		device    Device
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   string
	}{
		{
			name:      "empty actions is valid",
			device:    Device{Actions: ""},
			setupMock: func(mock sqlmock.Sqlmock) {},
			wantErr:   "",
		},
		{
			name:      "invalid JSON returns error",
			device:    Device{Actions: "not-json"},
			setupMock: func(mock sqlmock.Sqlmock) {},
			wantErr:   "actions must be a list of action IDs",
		},
		{
			name:      "invalid JSON array returns error",
			device:    Device{Actions: `{"foo": "bar"}`},
			setupMock: func(mock sqlmock.Sqlmock) {},
			wantErr:   "actions must be a list of action IDs",
		},
		{
			name:   "all actions exist is valid",
			device: Device{Actions: "[1, 2, 3]"},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery("SELECT EXISTS").WithArgs(2).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery("SELECT EXISTS").WithArgs(3).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			wantErr: "",
		},
		{
			name:   "non-existent action returns error",
			device: Device{Actions: "[1, 2, 999]"},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery("SELECT EXISTS").WithArgs(2).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery("SELECT EXISTS").WithArgs(999).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			wantErr: "all actions must exist",
		},
		{
			name:   "first action does not exist",
			device: Device{Actions: "[100]"},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").WithArgs(100).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			wantErr: "all actions must exist",
		},
		{
			name:   "database error is returned",
			device: Device{Actions: "[1]"},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").WithArgs(1).WillReturnError(errors.New("database connection failed"))
			},
			wantErr: "database connection failed",
		},
		{
			name:   "single valid action",
			device: Device{Actions: "[5]"},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").WithArgs(5).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close() // nolint

			tt.setupMock(mock)

			err = tt.device.ValidateWithDB(context.Background(), db)

			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
