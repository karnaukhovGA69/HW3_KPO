package gateway

import (
	"net/http"
	"time"
)

type Gateway struct {
	storageBaseURL  string
	analysisBaseURL string
	httpClient      *http.Client
}

func NewGateway(storageBaseURL, analysisBaseURL string) *Gateway {
	return &Gateway{
		storageBaseURL:  storageBaseURL,
		analysisBaseURL: analysisBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

