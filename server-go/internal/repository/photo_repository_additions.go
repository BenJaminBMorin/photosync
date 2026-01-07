package repository

import (
	"context"
	"database/sql"
	"strings"
)

// DeleteAll deletes all photos from the database
// Returns the number of photos deleted
func (r *PhotoRepository) DeleteAll(ctx context.Context) (int, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM photos")
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affected), nil
}

// VerifyExistence checks which photo IDs exist in the database
// Returns a map where keys are photo IDs and values indicate existence (true/false)
func (r *PhotoRepository) VerifyExistence(ctx context.Context, ids []string) (map[string]bool, error) {
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}

	// Initialize result map with all IDs as false
	result := make(map[string]bool, len(ids))
	for _, id := range ids {
		result[id] = false
	}

	// Build query with placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `SELECT id FROM photos WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Mark found IDs as true
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}

	return result, rows.Err()
}
