package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cleanInitialMigration(nhostFolder string) error {
	migrationDir := filepath.Join(nhostFolder, "migrations", "default")

	migrationRoot, err := os.OpenRoot(migrationDir)
	if err != nil {
		return fmt.Errorf("failed to open migrations directory %s: %w", migrationDir, err)
	}
	defer migrationRoot.Close()

	directory, err := migrationRoot.Open(".")
	if err != nil {
		return fmt.Errorf("failed to open migrations directory %s: %w", migrationDir, err)
	}
	defer directory.Close()

	migrations, err := directory.ReadDir(-1)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %s: %w", migrationDir, err)
	}

	for _, migration := range migrations {
		if !migration.IsDir() || !strings.HasSuffix(migration.Name(), "_init") ||
			!filepath.IsLocal(migration.Name()) {
			continue
		}

		upSQL := filepath.Join(migration.Name(), "up.sql")

		contents, err := migrationRoot.ReadFile(upSQL)
		if err != nil {
			return fmt.Errorf(
				"failed to read init migration %s: %w",
				filepath.Join(migrationDir, upSQL),
				err,
			)
		}

		cleaned := stripPsqlMetaCommands(contents)
		if bytes.Equal(contents, cleaned) {
			continue
		}

		if err := migrationRoot.WriteFile(upSQL, cleaned, 0o600); err != nil { //nolint:mnd
			return fmt.Errorf(
				"failed to write init migration %s: %w",
				filepath.Join(migrationDir, upSQL),
				err,
			)
		}
	}

	return nil
}

func stripPsqlMetaCommands(sql []byte) []byte {
	lines := bytes.SplitAfter(sql, []byte("\n"))

	cleaned := make([]byte, 0, len(sql))
	for _, line := range lines {
		fields := strings.Fields(string(line))
		if len(fields) > 0 && (fields[0] == `\restrict` || fields[0] == `\unrestrict`) {
			continue
		}

		cleaned = append(cleaned, line...)
	}

	return cleaned
}
