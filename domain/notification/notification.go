package notification

import "time"

type Notification struct {
	ID          string
	UserID      string
	Title       string
	Content     string
	ChannelName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateRequest struct {
	UserID      string            `json:"user_id"`
	Title       string            `json:"title" binding:"required"`
	Content     string            `json:"content" binding:"required"`
	ChannelName string            `json:"channel_name" binding:"required,oneof=email sms push"`
	Meta        map[string]string `json:"meta"`
}

type UpdateRequest struct {
	Title   string            `json:"title"`
	Content string            `json:"content"`
	Meta    map[string]string `json:"meta"`
}

type ListQuery struct {
	UserID    string
	Limit     int
	NextToken string
}

type ListResponse struct {
	Notifications []*Notification `json:"notifications"`
	NextToken     string          `json:"next_token"`
	HasMore       bool            `json:"has_more"`
}
