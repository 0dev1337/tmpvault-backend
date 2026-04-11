package upload

type UploadResponse struct {
	ID     string `json:"id,omitempty"`
	Expiry string `json:"expiry,omitempty"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
