package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripPsqlMetaCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "removes pg dump lock commands",
			input: "\\restrict hasura\nSET transaction_timeout = 0;\nSET check_function_bodies = false;\n\\unrestrict hasura\n",
			want:  "SET transaction_timeout = 0;\nSET check_function_bodies = false;\n",
		},
		{
			name:  "keeps other commands and similar tokens",
			input: "\\restrictx hasura\n\\unrestrictx hasura\n\\set ON_ERROR_STOP on\n-- \\restrict hasura\n",
			want:  "\\restrictx hasura\n\\unrestrictx hasura\n\\set ON_ERROR_STOP on\n-- \\restrict hasura\n",
		},
		{
			name:  "handles indentation and crlf",
			input: "  \\restrict hasura\r\nSELECT 1;\r\n\t\\unrestrict hasura\r\n",
			want:  "SELECT 1;\r\n",
		},
		{
			name:  "keeps empty input empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := string(stripPsqlMetaCommands([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("stripPsqlMetaCommands() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanInitialMigration(t *testing.T) {
	t.Parallel()

	nhostFolder := t.TempDir()
	migrationDir := filepath.Join(nhostFolder, "migrations", "default")
	initMigrationDir := filepath.Join(migrationDir, "20261014120000_init")
	otherMigrationDir := filepath.Join(migrationDir, "20261014120001_feature")

	for _, dir := range []string{initMigrationDir, otherMigrationDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create migration directory: %v", err)
		}
	}

	initSQL := []byte(
		"\\restrict hasura\nCREATE TABLE example (id integer);\n\\unrestrict hasura\n",
	)
	otherSQL := []byte("\\restrict hasura\nSELECT 1;\n\\unrestrict hasura\n")
	initUpSQL := filepath.Join(initMigrationDir, "up.sql")
	otherUpSQL := filepath.Join(otherMigrationDir, "up.sql")

	if err := os.WriteFile(initUpSQL, initSQL, 0o600); err != nil {
		t.Fatalf("write init migration: %v", err)
	}

	if err := os.WriteFile(otherUpSQL, otherSQL, 0o600); err != nil {
		t.Fatalf("write other migration: %v", err)
	}

	if err := cleanInitialMigration(nhostFolder); err != nil {
		t.Fatalf("clean initial migration: %v", err)
	}

	gotInitSQL, err := os.ReadFile(initUpSQL)
	if err != nil {
		t.Fatalf("read init migration: %v", err)
	}

	if got, want := string(gotInitSQL), "CREATE TABLE example (id integer);\n"; got != want {
		t.Errorf("cleaned init migration = %q, want %q", got, want)
	}

	gotOtherSQL, err := os.ReadFile(otherUpSQL)
	if err != nil {
		t.Fatalf("read other migration: %v", err)
	}

	if got := string(gotOtherSQL); got != string(otherSQL) {
		t.Errorf("other migration = %q, want %q", got, otherSQL)
	}
}
