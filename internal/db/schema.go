package db

import (
	"database/sql"
	"fmt"
	"os"
)

func InitSchema(database *sql.DB, shemaPath string) error {
	schema, err := os.ReadFile(shemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	if _, err := database.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}
