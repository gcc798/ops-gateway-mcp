package storage

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Open(ctx context.Context, path string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.ExecContext(ctx, `PRAGMA busy_timeout=5000`); err == nil {
		err = migrate(ctx, db, path)
	}
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize gateway database: %w", err)
	}
	return db, nil
}

func provider(db *sqlx.DB) (*goose.Provider, error) {
	files, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectSQLite3, db.DB, files,
		goose.WithDisableGlobalRegistry(true),
	)
}

func migrate(ctx context.Context, db *sqlx.DB, path string) error {
	p, err := provider(db)
	if err != nil {
		return err
	}
	current, target, err := p.GetVersions(ctx)
	if err != nil {
		return err
	}
	if current > target {
		return fmt.Errorf("database version %d exceeds supported version %d", current, target)
	}
	pending, err := p.HasPending(ctx)
	if err != nil || !pending {
		return err
	}
	var tables int
	if err := db.GetContext(ctx, &tables, `SELECT count(*) FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name<>'goose_db_version'`); err != nil {
		return err
	}
	if tables > 0 {
		if err := backup(ctx, db, path); err != nil {
			return err
		}
	}
	_, err = p.Up(ctx)
	return err
}

func backup(ctx context.Context, db *sqlx.DB, path string) error {
	// 通过 SQLite 生成一致性快照，避免直接复制文件时遗漏 WAL 中的数据。
	file, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".pre-migrate-*.db")
	if err != nil {
		return fmt.Errorf("create migration backup: %w", err)
	}
	name := file.Name()
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(name)
		}
	}()
	if err := file.Close(); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `VACUUM INTO ?`, name); err != nil {
		return fmt.Errorf("write migration backup: %w", err)
	}
	complete = true
	return nil
}
