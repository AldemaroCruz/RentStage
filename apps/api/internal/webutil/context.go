package webutil

import "context"

type contextKey string

const (
	tenantIDKey    contextKey = "tenant_id"
	requestIDKey   contextKey = "request_id"
	actorIDKey     contextKey = "actor_id"
	userIDKey      contextKey = "user_id"
	identityUIDKey contextKey = "identity_uid"
	userEmailKey   contextKey = "user_email"
	userNameKey    contextKey = "user_name"
	roleKey        contextKey = "role"
)

func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

func TenantID(ctx context.Context) string {
	value, _ := ctx.Value(tenantIDKey).(string)
	return value
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func WithActorID(ctx context.Context, actorID string) context.Context {
	return context.WithValue(ctx, actorIDKey, actorID)
}

func ActorID(ctx context.Context) string {
	value, _ := ctx.Value(actorIDKey).(string)
	return value
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserID(ctx context.Context) string {
	value, _ := ctx.Value(userIDKey).(string)
	return value
}

func WithIdentityUID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, identityUIDKey, uid)
}

func IdentityUID(ctx context.Context) string {
	value, _ := ctx.Value(identityUIDKey).(string)
	return value
}

func WithUserEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, userEmailKey, email)
}

func UserEmail(ctx context.Context) string {
	value, _ := ctx.Value(userEmailKey).(string)
	return value
}

func WithUserName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, userNameKey, name)
}

func UserName(ctx context.Context) string {
	value, _ := ctx.Value(userNameKey).(string)
	return value
}

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

func Role(ctx context.Context) string {
	value, _ := ctx.Value(roleKey).(string)
	return value
}
