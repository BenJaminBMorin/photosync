package models

import "time"

// ConfigCategory represents the category of a configuration item
type ConfigCategory string

const (
	CategoryServer   ConfigCategory = "server"
	CategoryDatabase ConfigCategory = "database"
	CategoryStorage  ConfigCategory = "storage"
	CategoryEmail    ConfigCategory = "email"
	CategorySecurity ConfigCategory = "security"
)

// ConfigItem represents a single configuration item
type ConfigItem struct {
	Key             string         `json:"key"`
	Value           string         `json:"value"`
	ValueType       string         `json:"valueType"` // string, int, bool, encrypted
	Category        ConfigCategory `json:"category"`
	RequiresRestart bool           `json:"requiresRestart"`
	IsSensitive     bool           `json:"isSensitive"` // Hide value in UI
	Description     string         `json:"description,omitempty"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// ConfigResponse is the response format for getting all configuration
type ConfigResponse struct {
	Items           []ConfigItem `json:"items"`
	RestartRequired bool         `json:"restartRequired"`
}

// UpdateConfigRequest is the request body for updating configuration
type UpdateConfigRequest struct {
	Updates []ConfigUpdate `json:"updates"`
}

// ConfigUpdate represents a single configuration update
type ConfigUpdate struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SMTPConfig represents SMTP server configuration
type SMTPConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"` // Only for input, never returned
	FromAddress string `json:"fromAddress"`
	FromName    string `json:"fromName"`
	UseTLS      bool   `json:"useTls"`
	SkipVerify  bool   `json:"skipVerify"` // For self-signed certs
}

// TestEmailRequest is the request body for sending a test email
type TestEmailRequest struct {
	Email string `json:"email"`
}

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid              bool     `json:"valid"`
	MissingItems       []string `json:"missingItems"`
	DatabaseOK         bool     `json:"databaseOk"`
	StorageOK          bool     `json:"storageOk"`
	EmailConfigured    bool     `json:"emailConfigured"`
	FirebaseConfigured bool     `json:"firebaseConfigured"`
}
