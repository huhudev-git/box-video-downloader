package models

// Info file info
type Info struct {
	FileVersion              FileVersion `json:"file_version"`
	AuthenticatedDownloadURL string      `json:"authenticated_download_url"`
	IsDownloadAvailable      bool        `json:"is_download_available"`
	Name                     string      `json:"name"`
}
