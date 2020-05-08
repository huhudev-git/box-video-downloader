package models

// FileVersion file version info
type FileVersion struct {
	ID   string `json:"id"`
	Sha1 string `json:"sha1"`
	Type bool   `json:"type"`
}
