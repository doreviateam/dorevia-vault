package models

import (
	"time"

	"github.com/google/uuid"
)

// Document représente un document stocké dans le système
type Document struct {
	ID          uuid.UUID `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	SHA256Hex   string    `json:"sha256_hex"`
	StoredPath  string    `json:"stored_path"`
	CreatedAt   time.Time `json:"created_at"`
}

// DocumentListResponse représente la réponse pour la liste de documents
type DocumentListResponse struct {
	Data       []Document        `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse contient les informations de pagination
type PaginationResponse struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
	Pages int `json:"pages"`
}

// DocumentQuery représente les paramètres de requête pour la recherche
type DocumentQuery struct {
	Page      int
	Limit     int
	Search    string
	Type      string
	DateFrom  *time.Time
	DateTo    *time.Time
}

