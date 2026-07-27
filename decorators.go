package microscope

import "context"

// EventPublisher publishes domain events.
type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload map[string]any) error
}

// OTPNotifier delivers one-time password codes.
type OTPNotifier interface {
	SendSignupOTP(ctx context.Context, email, otp string) error
	SendPasswordResetOTP(ctx context.Context, email, otp string) error
	SendEmailChangeOTP(ctx context.Context, email, otp string) error
}

type eventPublisherDecorator struct {
	inner EventPublisher
	hub   *Hub
}

func (p *eventPublisherDecorator) Publish(ctx context.Context, eventType string, payload map[string]any) error {
	if p.hub != nil {
		p.hub.RecordEvent(ctx, eventType, payload)
	}
	if p.inner == nil {
		return nil
	}
	return p.inner.Publish(ctx, eventType, payload)
}

// WrapEventPublisher records published events through hub.
func WrapEventPublisher(hub *Hub, inner EventPublisher) EventPublisher {
	if hub == nil || inner == nil {
		return inner
	}
	return &eventPublisherDecorator{inner: inner, hub: hub}
}

type otpNotifierDecorator struct {
	inner OTPNotifier
	hub   *Hub
}

func (n *otpNotifierDecorator) SendSignupOTP(ctx context.Context, email, otp string) error {
	n.record(ctx, "signup_otp", email)
	if n.inner == nil {
		return nil
	}
	return n.inner.SendSignupOTP(ctx, email, otp)
}

func (n *otpNotifierDecorator) SendPasswordResetOTP(ctx context.Context, email, otp string) error {
	n.record(ctx, "password_reset_otp", email)
	if n.inner == nil {
		return nil
	}
	return n.inner.SendPasswordResetOTP(ctx, email, otp)
}

func (n *otpNotifierDecorator) SendEmailChangeOTP(ctx context.Context, email, otp string) error {
	n.record(ctx, "email_change_otp", email)
	if n.inner == nil {
		return nil
	}
	return n.inner.SendEmailChangeOTP(ctx, email, otp)
}

func (n *otpNotifierDecorator) record(ctx context.Context, kind, email string) {
	if n.hub == nil {
		return
	}
	n.hub.RecordNotification(ctx, kind, map[string]any{
		"email": email,
		"otp":   "[REDACTED]",
	})
}

// WrapOTPNotifier records OTP notifications through hub without storing codes.
func WrapOTPNotifier(hub *Hub, inner OTPNotifier) OTPNotifier {
	if hub == nil || inner == nil {
		return inner
	}
	return &otpNotifierDecorator{inner: inner, hub: hub}
}

// WrapFunc decorates inner with a recording hook. Returns inner when hub is nil.
func WrapFunc[T any](hub *Hub, inner T, wrap func(hub *Hub, inner T) T) T {
	if hub == nil {
		return inner
	}
	return wrap(hub, inner)
}
