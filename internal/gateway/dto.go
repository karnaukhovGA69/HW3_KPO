package gateway

// DTO из storage-сервиса
type Work struct {
	ID         int64  `json:"id"`
	Student    string `json:"student"`
	Task       string `json:"task"`
	FilePath   string `json:"file_path"`
	UploadedAt string `json:"uploaded_at"`
}

// DTO из analysis-сервиса
type Report struct {
	ID         int64   `json:"id"`
	WorkID     int64   `json:"work_id"`
	Status     string  `json:"status"`
	Similarity float64 `json:"similarity"`
	Details    string  `json:"details"`
	CreatedAt  string  `json:"created_at"`
}

// тело запроса на создание работы через gateway
type CreateWorkRequest struct {
	Student  string `json:"student"`
	Task     string `json:"task"`
	FilePath string `json:"file_path"`
}

// объединённый ответ: работа + отчёт
type CombinedWorkResponse struct {
	Work   Work   `json:"work"`
	Report Report `json:"report"`
}

