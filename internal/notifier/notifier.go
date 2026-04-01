package notifier

import "context"

type Notifier interface {
Send(ctx context.Context, userId int64, content string) error
}
