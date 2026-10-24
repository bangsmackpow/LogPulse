package db

import "time"

type DiscoveredLog struct {
	ID           int64     `json:"id"`
	FilePath     string    `json:"filePath"`
	AppName      string    `json:"appName"`
	FileSize     int64     `json:"fileSize"`
	LastModified time.Time `json:"lastModified"`
	Format       string    `json:"format"`
	IsFavorite   bool      `json:"isFavorite"`
	Tag          string    `json:"tag"`
}

type AIAccountingEntry struct {
	ID           int64     `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"inputTokens"`
	OutputTokens int       `json:"outputTokens"`
	CostUSD      float64   `json:"costUsd"`
	QuerySummary string    `json:"querySummary"`
}
