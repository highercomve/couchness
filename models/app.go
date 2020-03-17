package models

// AppConfiguration couchness app configuration
type AppConfiguration struct {
	Name             string   `json:"name,omitempty"`
	MediaDir         string   `json:"media_dir,omitempty"`
	MediaDirs        []string `json:"media_directories,omitempty"`
	OmdbAPIKey       string   `json:"omdb_api_key"`
	TransmissionHost string   `json:"transmission_host"`
	TransmissionPort string   `json:"transmission_port"`
	TransmissionAuth string   `json:"transmission_auth"`
}
