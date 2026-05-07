package config

import "os"

// GetBaseURL returns the base URL from environment variable with fallback
func GetBaseURL() string {
	if baseURL := os.Getenv("base_url"); baseURL != "" {
		return baseURL
	}
	return ""
}

var (
	GET_RUNNING_SYSTEM_CONFIG_ENDPOINT string
)

func Initialize() {
	baseURL := GetBaseURL()
	GET_RUNNING_SYSTEM_CONFIG_ENDPOINT = baseURL + "/api/document/system-config/get-running-config"
}

// HTTP Configuration
const (
	CONTENT_TYPE_JSON = "application/json"
)
