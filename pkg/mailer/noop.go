package mailer

import (
	"context"
	"log/slog"
)

type NoopMailer struct{}

func (NoopMailer) Send(ctx context.Context, msg Message) error {
	slog.InfoContext(ctx, "mailer noop (email not sent)",
		slog.String("to", msg.To),
		slog.String("subject", msg.Subject),
	)
	return nil
}
