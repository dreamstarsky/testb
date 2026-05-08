package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"smog-detector/internal/model"
)

var ErrNotFound = errors.New("not found")

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(path string) (*SQLiteRepository, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir data dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	repo := &SQLiteRepository{db: db}
	if err := repo.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return repo, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

func (r *SQLiteRepository) SaveClient(ctx context.Context, clientID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO clients (client_id, created_at, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(client_id) DO UPDATE SET updated_at = excluded.updated_at
	`, clientID, time.Now().UTC(), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("save client: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) SaveLocation(ctx context.Context, record model.ClientLocation) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO client_locations (client_id, city_name, location_id, lat, lon, adm1, adm2, source, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_id) DO UPDATE SET
			city_name = excluded.city_name,
			location_id = excluded.location_id,
			lat = excluded.lat,
			lon = excluded.lon,
			adm1 = excluded.adm1,
			adm2 = excluded.adm2,
			source = excluded.source,
			updated_at = excluded.updated_at
	`, record.ClientID, record.City.Name, record.City.LocationID, record.City.Lat, record.City.Lon, record.City.Adm1, record.City.Adm2, record.Source, record.UpdatedAt.UTC())
	if err != nil {
		return fmt.Errorf("save location: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) GetLocation(ctx context.Context, clientID string) (model.ClientLocation, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT client_id, city_name, location_id, lat, lon, adm1, adm2, source, updated_at
		FROM client_locations
		WHERE client_id = ?
	`, clientID)

	var record model.ClientLocation
	var updatedAt time.Time
	if err := row.Scan(&record.ClientID, &record.City.Name, &record.City.LocationID, &record.City.Lat, &record.City.Lon, &record.City.Adm1, &record.City.Adm2, &record.Source, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ClientLocation{}, ErrNotFound
		}
		return model.ClientLocation{}, fmt.Errorf("get location: %w", err)
	}
	record.UpdatedAt = updatedAt
	return record, nil
}

func (r *SQLiteRepository) SaveDashboard(ctx context.Context, clientID string, dashboard model.Dashboard, fetchedAt, expiresAt time.Time) error {
	payload, err := json.Marshal(dashboard)
	if err != nil {
		return fmt.Errorf("marshal dashboard: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO weather_snapshots (client_id, payload_json, fetched_at, expires_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(client_id) DO UPDATE SET
			payload_json = excluded.payload_json,
			fetched_at = excluded.fetched_at,
			expires_at = excluded.expires_at
	`, clientID, string(payload), fetchedAt.UTC(), expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("save dashboard: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) GetDashboard(ctx context.Context, clientID string) (model.CachedDashboard, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT payload_json, fetched_at, expires_at
		FROM weather_snapshots
		WHERE client_id = ?
	`, clientID)

	var payload string
	var fetchedAt time.Time
	var expiresAt time.Time
	if err := row.Scan(&payload, &fetchedAt, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.CachedDashboard{}, ErrNotFound
		}
		return model.CachedDashboard{}, fmt.Errorf("get dashboard: %w", err)
	}

	var dashboard model.Dashboard
	if err := json.Unmarshal([]byte(payload), &dashboard); err != nil {
		return model.CachedDashboard{}, fmt.Errorf("unmarshal dashboard: %w", err)
	}

	return model.CachedDashboard{
		Dashboard: dashboard,
		FetchedAt: fetchedAt,
		ExpiresAt: expiresAt,
	}, nil
}

func (r *SQLiteRepository) initSchema() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS clients (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS client_locations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id TEXT NOT NULL UNIQUE,
			city_name TEXT NOT NULL,
			location_id TEXT NOT NULL,
			lat REAL NOT NULL,
			lon REAL NOT NULL,
			adm1 TEXT,
			adm2 TEXT,
			source TEXT NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY(client_id) REFERENCES clients(client_id)
		);`,
		`CREATE TABLE IF NOT EXISTS weather_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id TEXT NOT NULL UNIQUE,
			payload_json TEXT NOT NULL,
			fetched_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY(client_id) REFERENCES clients(client_id)
		);`,
	}

	for _, stmt := range statements {
		if _, err := r.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}
