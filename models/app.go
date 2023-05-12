package models

// AppConfiguration couchness app configuration
type AppConfiguration struct {
	Name             string   `json:"name,omitempty"`
	ShowsDir         string   `json:"media_dir,omitempty"`
	ShowsDirs        []string `json:"media_directories,omitempty"`
	MoviesDir        string   `json:"movies_dir,omitempty"`
	OmdbAPIKey       string   `json:"omdb_api_key"`
	TransmissionHost string   `json:"transmission_host"`
	TransmissionPort string   `json:"transmission_port"`
	TransmissionAuth string   `json:"transmission_auth"`
}
