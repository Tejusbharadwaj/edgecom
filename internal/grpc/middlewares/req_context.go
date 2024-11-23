package middleware

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type contextKey string

const requestIDKey contextKey = "requestID"

func ContextMiddleware(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	ctx = context.WithValue(ctx, requestIDKey, generateRequestID())
	return handler(ctx, req)
}

func generateRequestID() string {
	return uuid.NewString()
}
