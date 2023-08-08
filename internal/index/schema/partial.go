package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Partial struct {
	ID        int64
	PackageID int64
	Parts     string
	NumParts  int
}

func SyncPartials(ctx context.Context, db Querier, partials []Partial) (rerr error) {
	stmt, err := db.PrepareContext(ctx, `
INSERT INTO partial (packageId, parts) VALUES (?, ?);
`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close statement: %w", err))
		}
	}()

	for _, partial := range partials {
		_, err := stmt.ExecContext(ctx, partial.PackageID, partial.Parts)
		if err != nil {
			return fmt.Errorf("failed to execute prepared statement: %w", err)
		}
	}
	return nil
}

func selectPartials(ctx context.Context, db Querier, partials []Partial) (_ []Partial, rerr error) {
	rows, err := db.QueryContext(ctx, `
SELECT rowid, packageId, parts, numParts FROM partial;
`)
	if err != nil {
		return nil, fmt.Errorf("failed to select partials: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close rows: %w", err))
		}
	}()
	return scanPartials(ctx, rows, partials)
}

func scanPartials(ctx context.Context, rows *sql.Rows, partials []Partial) ([]Partial, error) {
	for rows.Next() && rows.Err() == nil {
		partial, err := scanPartial(rows)
		if err != nil {
			return partials, fmt.Errorf("failed to scan partial: %w", err)
		}
		partials = append(partials, partial)
	}
	if err := rows.Err(); err != nil {
		return partials, fmt.Errorf("failed to load next partial: %w", err)
	}
	return partials, nil
}

func scanPartial(row Scanner) (partial Partial, _ error) {
	return partial, row.Scan(&partial.ID, &partial.PackageID, &partial.Parts, &partial.NumParts)
}
