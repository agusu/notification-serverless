package notification

import "context"

type Queue interface {
	Publish(ctx context.Context, message *DispatchMessage) error
}
