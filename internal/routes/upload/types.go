package upload

type UploadResponse struct {
	URL    string `json:"url,omitempty"`
	Expiry string `json:"expiry,omitempty"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
