package audit

import "context"

type requestIDKey struct{}
type toolKey struct{}

func WithTool(ctx context.Context, tool string) context.Context {
	return context.WithValue(ctx, toolKey{}, tool)
}
func Tool(ctx context.Context) string { tool, _ := ctx.Value(toolKey{}).(string); return tool }

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}
func RequestID(ctx context.Context) string { id, _ := ctx.Value(requestIDKey{}).(string); return id }
