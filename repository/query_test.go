package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryRepo_GetIDByName(t *testing.T) {
	tests := []struct {
		name       string
		table      string
		lookupName string
		setupMock  func(mock sqlmock.Sqlmock)
		wantID     int
		wantErr    string
	}{
		{
			name:       "returns id when row found",
			table:      "devices",
			lookupName: "sensor-1",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM devices WHERE name = \\?").
					WithArgs("sensor-1").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
			},
			wantID: 42,
		},
		{
			name:       "returns error when row not found",
			table:      "devices",
			lookupName: "missing",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM devices WHERE name = \\?").
					WithArgs("missing").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: "looking up 'missing' in devices: sql: no rows in result set",
		},
		{
			name:       "returns error on db failure",
			table:      "actions",
			lookupName: "toggle",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM actions WHERE name = \\?").
					WithArgs("toggle").
					WillReturnError(fmt.Errorf("connection refused"))
			},
			wantErr: "looking up 'toggle' in actions: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close() // nolint

			tt.setupMock(mock)

			repo := NewQueryRepo(db, []string{"devices", "actions"})
			id, err := repo.GetIDByName(context.Background(), tt.table, tt.lookupName)

			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				assert.Equal(t, 0, id)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}

			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
