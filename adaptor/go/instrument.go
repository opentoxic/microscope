package microscope

import "context"

// RecordPublish records a domain event through hub.
func RecordPublish(hub *Hub, ctx context.Context, eventType string, payload map[string]any) {
	if hub == nil {
		return
	}
	hub.RecordEvent(ctx, eventType, payload)
}

// RecordOTP records an OTP notification.
func RecordOTP(hub *Hub, ctx context.Context, kind, email, otp string) {
	if hub == nil {
		return
	}
	hub.RecordNotification(ctx, kind, map[string]any{
		"email": email,
		"otp":   hub.SanitizeOTP(otp),
	})
}

// WrapPublishFunc returns a closure that records then delegates to inner.
func (i *Integration) WrapPublishFunc(inner func(ctx context.Context, eventType string, payload map[string]any) error) func(ctx context.Context, eventType string, payload map[string]any) error {
	if i == nil || i.hub == nil || inner == nil {
		return inner
	}
	hub := i.hub
	return func(ctx context.Context, eventType string, payload map[string]any) error {
		RecordPublish(hub, ctx, eventType, payload)
		return inner(ctx, eventType, payload)
	}
}

// WrapOTPFunc returns a closure that records then delegates to inner for signup OTP.
func (i *Integration) WrapOTPFunc(kind string, inner func(ctx context.Context, email, otp string) error) func(ctx context.Context, email, otp string) error {
	if i == nil || i.hub == nil || inner == nil {
		return inner
	}
	hub := i.hub
	return func(ctx context.Context, email, otp string) error {
		RecordOTP(hub, ctx, kind, email, otp)
		return inner(ctx, email, otp)
	}
}
