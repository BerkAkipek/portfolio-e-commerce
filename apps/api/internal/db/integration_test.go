package db_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

type integrationEnv struct {
	dsn string
}

func setupIntegrationEnv(t *testing.T) integrationEnv {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping integration tests")
	}
	if _, err := exec.LookPath("psql"); err != nil {
		t.Skip("psql is not available; skipping integration tests")
	}

	env := integrationEnv{dsn: dsn}
	mustExecSQL(t, env, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`)
	for _, file := range migrationFiles(t) {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		mustExecSQL(t, env, string(content))
	}
	return env
}

func migrationFiles(t *testing.T) []string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve current file path")
	}

	migrationsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no migrations found in %s", migrationsDir)
	}
	return files
}

func mustExecSQL(t *testing.T, env integrationEnv, sql string) {
	t.Helper()
	if err := execSQL(env, sql); err != nil {
		t.Fatalf("sql failed: %v\nsql:\n%s", err, sql)
	}
}

func execSQL(env integrationEnv, sql string) error {
	cmd := exec.Command(
		"psql",
		env.dsn,
		"-v", "ON_ERROR_STOP=1",
		"-X",
		"-q",
		"-c", sql,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return &sqlExecError{cause: err, output: msg}
	}
	return nil
}

type sqlExecError struct {
	cause  error
	output string
}

func (e *sqlExecError) Error() string {
	return e.cause.Error() + ": " + e.output
}

func TestUsersEmailCaseInsensitiveUnique(t *testing.T) {
	env := setupIntegrationEnv(t)

	mustExecSQL(t, env, `
INSERT INTO users (id, email, full_name, role)
VALUES ('00000000-0000-0000-0000-000000000001', 'alice@example.com', 'Alice', 'user');
`)

	err := execSQL(env, `
INSERT INTO users (id, email, full_name, role)
VALUES ('00000000-0000-0000-0000-000000000002', 'Alice@Example.com', 'Alice 2', 'user');
`)
	if err == nil {
		t.Fatalf("expected unique violation for case-insensitive email")
	}
}

func TestCartsOneActivePerUser(t *testing.T) {
	env := setupIntegrationEnv(t)

	mustExecSQL(t, env, `
INSERT INTO users (id, email, full_name, role)
VALUES ('00000000-0000-0000-0000-000000000010', 'cart-owner@example.com', 'Cart Owner', 'user');
`)

	mustExecSQL(t, env, `
INSERT INTO carts (id, user_id, status)
VALUES ('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000010', 'active');
`)

	err := execSQL(env, `
INSERT INTO carts (id, user_id, status)
VALUES ('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000010', 'active');
`)
	if err == nil {
		t.Fatalf("expected unique violation for second active cart")
	}
}

func TestOrdersShippedRequiresShipping(t *testing.T) {
	env := setupIntegrationEnv(t)

	mustExecSQL(t, env, `
INSERT INTO users (id, email, full_name, role)
VALUES ('00000000-0000-0000-0000-000000000020', 'buyer@example.com', 'Buyer', 'user');
`)

	err := execSQL(env, `
INSERT INTO orders (id, user_id, status, currency, subtotal_cents, total_cents)
VALUES (
  '00000000-0000-0000-0000-000000000021',
  '00000000-0000-0000-0000-000000000020',
  'shipped',
  'USD',
  1000,
  1200
);
`)
	if err == nil {
		t.Fatalf("expected check violation when shipped order has no shipping fields")
	}
}

func TestPaymentsUniqueCheckoutSessionID(t *testing.T) {
	env := setupIntegrationEnv(t)

	mustExecSQL(t, env, `
INSERT INTO users (id, email, full_name, role)
VALUES ('00000000-0000-0000-0000-000000000030', 'payer@example.com', 'Payer', 'user');
`)

	mustExecSQL(t, env, `
INSERT INTO orders (id, user_id, status, currency, subtotal_cents, total_cents)
VALUES (
  '00000000-0000-0000-0000-000000000031',
  '00000000-0000-0000-0000-000000000030',
  'pending',
  'USD',
  2500,
  2500
);
`)

	mustExecSQL(t, env, `
INSERT INTO payments (
  id,
  order_id,
  provider,
  amount_cents,
  currency,
  status,
  checkout_session_id
) VALUES (
  '00000000-0000-0000-0000-000000000032',
  '00000000-0000-0000-0000-000000000031',
  'stripe',
  2500,
  'USD',
  'pending',
  'cs_test_123'
);
`)

	err := execSQL(env, `
INSERT INTO payments (
  id,
  order_id,
  provider,
  amount_cents,
  currency,
  status,
  checkout_session_id
) VALUES (
  '00000000-0000-0000-0000-000000000033',
  '00000000-0000-0000-0000-000000000031',
  'stripe',
  2500,
  'USD',
  'pending',
  'cs_test_123'
);
`)
	if err == nil {
		t.Fatalf("expected unique violation for duplicate checkout session id")
	}
}
