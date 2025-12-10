package storage

import "time"

type Work struct {
	ID         int64     `json:"id"`
	Student    string    `json:"student"`
	Task       string    `json:"task"`
	FilePath   string    `json:"file_path"`
	UploadedAt time.Time `json:"uploaded_at"`
}
