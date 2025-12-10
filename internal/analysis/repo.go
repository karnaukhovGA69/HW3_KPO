package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const SimilarityUnknown = -1.0

type Report struct {
	ID         int64     `json:"id"`
	WorkID     int64     `json:"work_id"`
	Status     string    `json:"status"`
	Similarity float64   `json:"similarity"`
	Details    string    `json:"details"`
	CreatedAt  time.Time `json:"created_at"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r Repository) CreateReport(ctx context.Context, report *Report) error {
	const query = `
	INSERT INTO reports (work_id, status, similarity, details)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at;`

	row := r.pool.QueryRow(ctx, query, report.WorkID, report.Status, report.Similarity, report.Details)
	if err := row.Scan(&report.ID, &report.CreatedAt); err != nil {
		return fmt.Errorf("failed to insert report: %w", err)
	}
	return nil
}

func (r Repository) GetReport(ctx context.Context, id int64) (*Report, error) {
	const query = `
    SELECT id, work_id, status, similarity, details, created_at 
    FROM reports 
    WHERE id = $1;`

	row := r.pool.QueryRow(ctx, query, id)
	var report Report
	if err := row.Scan(&report.ID, &report.WorkID, &report.Status, &report.Similarity, &report.Details, &report.CreatedAt); err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}
	return &report, nil

}
