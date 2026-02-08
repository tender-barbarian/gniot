package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type Querier interface {
	GetIDByName(ctx context.Context, table, name string) (int, error)
}

type QueryRepo struct {
	db            *sql.DB
	allowedTables map[string]bool
}

func NewQueryRepo(db *sql.DB, allowedTables []string) *QueryRepo {
	r := &QueryRepo{db: db, allowedTables: make(map[string]bool, len(allowedTables))}
	for _, allowedTable := range allowedTables {
		r.allowedTables[allowedTable] = true
	}
	return r
}

func (r *QueryRepo) GetIDByName(ctx context.Context, table, name string) (int, error) {
	var id int
	if !r.allowedTables[table] {
		return 0, fmt.Errorf("invalid table: %s", table)
	}

	err := r.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT id FROM %s WHERE name = ?", table), name,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("looking up '%s' in %s: %w", name, table, err)
	}
	return id, nil
}
