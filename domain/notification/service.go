package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotificationNotFound  = errors.New("notification not found")
	ErrInvalidChannel        = errors.New("invalid channel")
	ErrDuplicateNotification = errors.New("notification already exists")
)

// ChannelValidator validates channel metadata (email, sms, push)
type ChannelValidator interface {
	Validate(channelName string, meta map[string]string) error
}

// Service contains the business logic for notifications
type Service struct {
	repo      Repository
	queue     Queue
	validator ChannelValidator
}

// NewService creates a new instance of the service
func NewService(repo Repository, queue Queue, validator ChannelValidator) *Service {
	return &Service{
		repo:      repo,
		queue:     queue,
		validator: validator,
	}
}

// Create creates a new notification and queues it for processing
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Notification, error) {
	if err := s.validator.Validate(req.ChannelName, req.Meta); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidChannel, err)
	}

	now := time.Now()
	notification := &Notification{
		ID:          generateID(),
		UserID:      req.UserID,
		Title:       req.Title,
		Content:     req.Content,
		ChannelName: req.ChannelName,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	message := DispatchMessage{
		NotificationID: notification.ID,
		UserID:         notification.UserID,
		ChannelName:    req.ChannelName,
		Title:          req.Title,
		Content:        req.Content,
		Meta:           req.Meta,
		ScheduledAt:    req.ScheduledAt,
	}

	if err := s.queue.Publish(ctx, &message); err != nil {
		// TODO: Implement compensation (delete from DB?)
		return nil, fmt.Errorf("failed to enqueue: %w", err)
	}

	return notification, nil
}

// GetByID gets a notification by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Notification, error) {
	return s.repo.GetByID(ctx, id)
}

// List lists notifications with filters
func (s *Service) List(ctx context.Context, query ListQuery) ([]*Notification, error) {
	return s.repo.List(ctx, query)
}

// Update updates a notification
// Currently unused because notifications are sent immediately
// Will be used in the future for scheduled notifications
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) error {
	// 1. Verify that it exists
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 2. Validate metadata if provided
	if req.Meta != nil {
		if err := s.validator.Validate(notification.ChannelName, req.Meta); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidChannel, err)
		}
	}

	// 3. Prepare updates
	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	updates["updated_at"] = time.Now()

	// 4. Update
	return s.repo.Update(ctx, id, updates)
}

// Delete deletes a notification (soft delete)
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// DispatchMessage is the message that is sent to SQS
type DispatchMessage struct {
	NotificationID string            `json:"notification_id"`
	UserID         string            `json:"user_id"`
	ChannelName    string            `json:"channel_name"`
	Title          string            `json:"title"`
	Content        string            `json:"content"`
	Meta           map[string]string `json:"meta"`
	ScheduledAt    *time.Time        `json:"scheduled_at,omitempty"`
}

// generateID generates a unique ID
func generateID() string {
	return uuid.New().String()
}
