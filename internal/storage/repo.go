package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

func (r *Repository) CreateWork(ctx context.Context, work *Work) error {
	const query = `
	INSERT INTO works (student, task, file_path)
	VALUES ($1, $2, $3)
	RETURNING id, uploaded_at;`

	row := r.pool.QueryRow(ctx, query, work.Student, work.Task, work.FilePath)
	if err := row.Scan(&work.ID, &work.UploadedAt); err != nil {
		return fmt.Errorf("create work: %w", err)
	}
	return nil
}

func (r *Repository) GetWork(ctx context.Context, id int64) (*Work, error) {
	const query = `
	Select id, student, task, file_path, uploaded_at FROM works  WHERE id = $1;`

	row := r.pool.QueryRow(ctx, query, id)
	var w Work

	if err := row.Scan(&w.ID, &w.Student, &w.Task, &w.FilePath, &w.UploadedAt); err != nil {
		return nil, fmt.Errorf("get work: %w", err)
	}
	return &w, nil
}
