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

// MessageQueue es la interfaz para encolar mensajes (SQS)
// La implementación real estará en clients/sqs.go
type MessageQueue interface {
	Send(ctx context.Context, message interface{}) error
}

// ChannelValidator valida metadata de canales (email, sms, push)
type ChannelValidator interface {
	Validate(channelName string, meta map[string]string) error
}

// Service contiene la lógica de negocio de notificaciones
type Service struct {
	repo      Repository
	queue     MessageQueue
	validator ChannelValidator
}

// NewService crea una nueva instancia del servicio
// Dependency Injection: recibe todas las dependencias por parámetro
func NewService(repo Repository, queue MessageQueue, validator ChannelValidator) *Service {
	return &Service{
		repo:      repo,
		queue:     queue,
		validator: validator,
	}
}

// Create crea una nueva notificación y la encola para procesamiento
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
		ChannelName:    req.ChannelName,
		Title:          req.Title,
		Content:        req.Content,
		Meta:           req.Meta,
		ScheduledAt:    req.ScheduledAt,
	}

	if err := s.queue.Send(ctx, message); err != nil {
		// TODO: Implementar compensación (eliminar de DB?)
		return nil, fmt.Errorf("failed to enqueue: %w", err)
	}

	return notification, nil
}

// GetByID obtiene una notificación por ID
func (s *Service) GetByID(ctx context.Context, id string) (*Notification, error) {
	return s.repo.GetByID(ctx, id)
}

// List lista notificaciones con filtros
func (s *Service) List(ctx context.Context, query ListQuery) ([]*Notification, error) {
	return s.repo.List(ctx, query)
}

// Update actualiza una notificación
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) error {
	// 1. Verificar que existe
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 2. Validar metadata si se proporciona
	if req.Meta != nil {
		if err := s.validator.Validate(notification.ChannelName, req.Meta); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidChannel, err)
		}
	}

	// 3. Preparar updates
	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	updates["updated_at"] = time.Now()

	// 4. Actualizar
	return s.repo.Update(ctx, id, updates)
}

// Delete elimina una notificación (soft delete)
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// DispatchMessage es el mensaje que se envía a SQS
type DispatchMessage struct {
	NotificationID string            `json:"notification_id"`
	ChannelName    string            `json:"channel_name"`
	Title          string            `json:"title"`
	Content        string            `json:"content"`
	Meta           map[string]string `json:"meta"`
	ScheduledAt    *time.Time        `json:"scheduled_at,omitempty"`
}

// generateID genera un ID único (puedes usar ULID para sortability)
func generateID() string {
	return uuid.New().String()
}
