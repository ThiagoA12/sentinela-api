//go:build integration

package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	testMigratorPassword = "migrator-de-teste"
	testRWPassword       = "rw-de-teste"
)

func connectAs(t *testing.T, user, password string) *pgx.Conn {
	t.Helper()
	cfg := testPool.Config().ConnConfig
	cfg.User = user
	cfg.Password = password
	conn, err := pgx.ConnectConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("conectar como %s: %v", user, err)
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return conn
}

func TestAppRWPermissions(t *testing.T) {
	conn := connectAs(t, "app_rw", testRWPassword)

	var timeout string
	if err := conn.QueryRow(t.Context(), "SHOW statement_timeout").Scan(&timeout); err != nil {
		t.Fatalf("ler statement_timeout: %v", err)
	}
	if timeout != "5s" {
		t.Errorf("statement_timeout = %q, quero 5s", timeout)
	}

	permitido := []string{
		"SELECT count(*) FROM assets",
		"SELECT count(*) FROM users",
		"SELECT count(*) FROM audit_log",
		"INSERT INTO audit_log (action, entity, request_id) VALUES ('login_failed', 'users', 'teste')",
	}
	for _, sql := range permitido {
		if _, err := conn.Exec(t.Context(), sql); err != nil {
			t.Errorf("%s: %v", sql, err)
		}
	}

	negado := []string{
		"UPDATE audit_log SET action = 'login'",
		"DELETE FROM audit_log",
		"TRUNCATE audit_log",
		"DELETE FROM assets",
		"DELETE FROM users",
		"CREATE TABLE intrusa (id int)",
	}
	for _, sql := range negado {
		_, err := conn.Exec(t.Context(), sql)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Errorf("%s: erro = %v, quero 42501 (insufficient_privilege)", sql, err)
		}
	}
}
