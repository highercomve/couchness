package models

// FollowService follow service definition
type FollowService interface {
	GetID() string
	GetURL() string
	ShowURL(showID string, page int, limit int) string
	GetShowData(show *Show, page int, limit int, typeOf string) (*Show, error)
}
