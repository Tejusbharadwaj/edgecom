package middleware

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	// Get request ID from context
	requestID, _ := ctx.Value(requestIDKey).(string)

	// Execute the handler
	resp, err := handler(ctx, req)

	// Log the request with request ID
	log.Printf(
		"request_id: %s method: %s duration: %s error: %v",
		requestID,
		info.FullMethod,
		time.Since(start),
		err,
	)

	return resp, err
}
