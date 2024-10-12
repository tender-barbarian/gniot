package repository

import (
	"context"
	"database/sql"
	"sync"

	sq "github.com/Masterminds/squirrel"
	"github.com/tender-barbarian/gniot/webserver/internal/model"
)

// SensorRepository is the repository to handle the [model.Sensor] model database interactions.
type SensorRepository struct {
	mutex sync.Mutex
	db    *sql.DB
}

// NewSensorRepository returns a new [SensorRepository].
func NewSensorRepository(db *sql.DB) *SensorRepository {
	return &SensorRepository{
		db: db,
	}
}

// Find finds a Sensor by id.
func (r *SensorRepository) Find(ctx context.Context, id int) (model.Sensor, error) {
	var Sensor model.Sensor

	query, args, err := sq.
		Select("id", "name", "sensorType", "Chip", "Board", "SensorMethodIDs").
		From("Sensors").
		Where(sq.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return Sensor, err
	}

	row := r.db.QueryRowContext(ctx, query, args...)

	err = row.Scan(&Sensor.ID, &Sensor.Name, &Sensor.SensorType, &Sensor.Chip, &Sensor.Board, &Sensor.SensorMethodIDs)

	return Sensor, err
}

// SensorRepositoryFindAllParams is a parameter for FindAll.
type SensorRepositoryFindAllParams struct {
	Name            sql.NullString
	SensorType      sql.NullString
	Chip            sql.NullString
	Board           sql.NullString
	SensorMethodIDs []int32
}

// FindAll finds all Sensors.
func (r *SensorRepository) FindAll(ctx context.Context, params SensorRepositoryFindAllParams) ([]model.Sensor, error) {
	qb := sq.
		Select("id", "name", "sensorType", "chip", "board", "SensorMethodIDs").
		From("Sensors")

	if params.Name.Valid {
		qb = qb.Where(sq.Eq{"name": params.Name.String})
	}

	if params.SensorType.Valid {
		qb = qb.Where(sq.Eq{"sensorType": params.SensorType.String})
	}

	if params.Chip.Valid {
		qb = qb.Where(sq.Eq{"chip": params.Chip.String})
	}

	if params.Board.Valid {
		qb = qb.Where(sq.Eq{"board": params.Board.String})
	}

	qb = qb.OrderBy("id")

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var Sensors []model.Sensor
	for rows.Next() {
		var Sensor model.Sensor
		if err = rows.Scan(&Sensor.ID, &Sensor.Name, &Sensor.SensorType, &Sensor.Chip, &Sensor.Board, &Sensor.SensorMethodIDs); err != nil {
			return nil, err
		}
		Sensors = append(Sensors, Sensor)
	}

	if err = rows.Close(); err != nil {
		return nil, err
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return Sensors, nil
}

// SensorRepositoryCreateParams is a parameter for Create.
type SensorRepositoryCreateParams struct {
	Name            string
	SensorType      string
	Chip            string
	Board           string
	SensorMethodIDs []int32
}

// Create creates a new Sensor and returns its id.
func (r *SensorRepository) Create(ctx context.Context, params SensorRepositoryCreateParams) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	query, args, err := sq.Insert("Sensors").Columns("name", "sensorType", "chip", "board", "sensorMethodIds").Values(params.Name, params.SensorType, params.Chip, params.Board, params.SensorMethodIDs).ToSql()
	if err != nil {
		return 0, err
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// Delete deletes an existing Sensor by id.
func (r *SensorRepository) Delete(ctx context.Context, id int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	query, args, err := sq.Delete("Sensors").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query, args...)

	return err
}
