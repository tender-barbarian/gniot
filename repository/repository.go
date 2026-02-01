package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	gocrud "github.com/tender-barbarian/go-crud"
)

type GenericRepo[M gocrud.Model] interface {
	Create(ctx context.Context, model M) (int, error)
	Get(ctx context.Context, id int) (M, error)
	GetAll(ctx context.Context) ([]M, error)
	Delete(ctx context.Context, id int) error
	Update(ctx context.Context, model M, id int) error
	GetTable() string
}

func NewDBConnection(dbPath, migrationsPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %v", err)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("creating driver: %v", err)
	}

	if migrationsPath == "" {
		migrationsPath = "file://db/migrations"
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "sqlite3", driver)
	if err != nil {
		return nil, fmt.Errorf("initialising migrations: %v", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("running migrations: %v", err)
	}

	return db, nil
}
