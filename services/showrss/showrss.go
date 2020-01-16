package showrss

import "github.com/highercomve/couchness/models"

const (
	//ServiceType showrss service type
	ServiceType = "showrss"
)

// Service show rss service
type Service struct {
	ID      string `json:"id"`
	BaseURL string `json:"base_url"`
}

// GetID get service ID
func (s Service) GetID() string {
	return s.ID
}

// GetURL get service base URL
func (s Service) GetURL() string {
	return s.BaseURL
}

// ShowURL get show information URL
func (s Service) ShowURL(showID string, page, limit int) string {
	return s.BaseURL + showID + ".rss"
}

// GetShowData get show data from service
func (s Service) GetShowData(show *models.Show, page, limit int) (*models.Show, error) {
	return nil, nil
}

// New create new show rss follow service
func New() Service {
	return Service{
		ID:      string(ServiceType),
		BaseURL: "https://showrss.info/show/",
	}
}
