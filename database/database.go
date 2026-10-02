package database

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lucap9056/auth-middleware/database/v2/schema"
)

const schemaAdvisoryLockKey int64 = 0x6175746864657669

type Database struct {
	db     *sql.DB
	ctx    context.Context
	cancel context.CancelFunc
}

func NewDatabase(dsn string, opts ...Option) (*Database, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	cfg := defaultOptions()
	for _, opt := range opts {
		opt(cfg)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)

	db.SetMaxIdleConns(cfg.MaxIdleConns)

	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if cfg.AutoCreateSchema {
		if err := createSchema(db, cfg.Schema); err != nil {
			db.Close()
			return nil, err
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	d := &Database{
		db:     db,
		ctx:    ctx,
		cancel: cancel,
	}

	if cfg.CleanupInterval > 0 {
		go d.startCleanupWorker(cfg.CleanupInterval)
	}

	return d, nil
}

func createSchema(db *sql.DB, params *schema.Params) error {
	schemaSQL, err := schema.Generate(params)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", schemaAdvisoryLockKey); err != nil {
		return err
	}
	if _, err := tx.Exec(schemaSQL); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *Database) cleanupOldDevices() (int64, error) {
	query := `
    DELETE FROM auth_user_devices 
    WHERE updated_at < NOW() - INTERVAL '7 days';
    `
	result, err := d.db.Exec(query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (d *Database) startCleanupWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, err := d.cleanupOldDevices()
			if err != nil {
				log.Println("Cleanup error:", err.Error())
			}
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *Database) Close() error {
	d.cancel()
	return d.db.Close()
}
