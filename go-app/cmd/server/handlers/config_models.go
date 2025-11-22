package handlers

// ConfigExportResponse represents the HTTP response for GET /api/v2/config
type ConfigExportResponse struct {
	Status string      `json:"status"`          // "success" or "error"
	Data   *ConfigData `json:"data,omitempty"`  // Response data (success only)
	Error  string      `json:"error,omitempty"` // Error message (error only)
}

// ConfigData contains configuration export data
type ConfigData struct {
	Version        string                 `json:"version"`                  // SHA256 hash
	Source         string                 `json:"source"`                   // Configuration source
	LoadedAt       string                 `json:"loaded_at"`                // RFC3339 timestamp
	ConfigFilePath string                 `json:"config_file_path,omitempty"` // Path if from file
	Config         map[string]interface{} `json:"config"`                    // Actual config data
}
