package notification

import "context"

// Repository define el contrato (interface) que debe cumplir cualquier implementación
// Esto permite cambiar DynamoDB por otra DB sin tocar el dominio
type Repository interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id string) (*Notification, error)
	List(ctx context.Context, query ListQuery) (*ListResponse, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}
