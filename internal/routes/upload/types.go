package upload

type UploadResponse struct {
	ID     string `json:"id,omitempty"`
	Expiry string `json:"expiry,omitempty"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// InitUploadResponse is returned by InitUploadHandler to start a chunked
// upload: the client must send every chunk at exactly ChunkSize bytes
// (except the last, which is whatever remains) to session_id/chunk?index=N,
// then POST session_id/complete.
type InitUploadResponse struct {
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	ChunkSize   int64  `json:"chunk_size,omitempty"`
	TotalChunks int    `json:"total_chunks,omitempty"`
}
