package models

// Info file info
type Info struct {
	FileVersion              FileVersion `json:"file_version"`
	AuthenticatedDownloadURL string      `json:"authenticated_download_url"`
	IsDownloadAvailable      bool        `json:"is_download_available"`
	Name                     string      `json:"name"`
	ID                       string      `json:"id"`
	Etag                     string      `json:"etag"`
	Extension                string      `json:"extension"`
	Size                     int         `json:"size"`
	Type                     string      `json:"type"`
}
