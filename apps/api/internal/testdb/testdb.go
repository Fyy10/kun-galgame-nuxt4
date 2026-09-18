package testdb

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const EnvVar = "TEST_DATABASE_DSN"

func Open(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv(EnvVar)
	if dsn == "" {
		if os.Getenv("KUN_REQUIRE_TEST_DB") == "1" {
			t.Fatalf("%s not set while KUN_REQUIRE_TEST_DB=1", EnvVar)
		}
		t.Skipf("%s not set — DB-backed test skipped", EnvVar)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test database: %v", redact(err.Error(), dsn))
	}
	return db
}

func redact(msg, dsn string) string {
	msg = strings.ReplaceAll(msg, dsn, "<"+EnvVar+">")
	for _, part := range dsnParts(dsn) {
		msg = strings.ReplaceAll(msg, part, "<redacted>")
	}
	return msg
}

func dsnParts(dsn string) []string {
	var out []string
	add := func(s string) {
		if len(s) >= 3 {
			out = append(out, s)
		}
	}

	if u, err := url.Parse(dsn); err == nil && u.Host != "" {
		add(u.Hostname())
		add(strings.TrimPrefix(u.Path, "/"))
		if u.User != nil {
			add(u.User.Username())
			if pw, ok := u.User.Password(); ok {
				add(pw)
			}
		}
	}
	for field := range strings.FieldsSeq(dsn) {
		if k, v, ok := strings.Cut(field, "="); ok {
			switch k {
			case "host", "user", "password", "dbname":
				add(v)
			}
		}
	}
	return out
}
