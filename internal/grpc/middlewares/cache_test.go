package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// Mock handler to simulate gRPC handler behavior.
func mockHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return "response-" + req.(string), nil
}

func TestCachingInterceptor(t *testing.T) {
	// Initialize the cache with a size of 2.
	err := InitializeCache(2)
	assert.NoError(t, err, "Failed to initialize cache")

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	//cache miss
	req1 := "request1"
	resp, err := CachingInterceptor(ctx, req1, info, mockHandler)
	assert.NoError(t, err, "Error in first request")
	assert.Equal(t, "response-request1", resp, "Unexpected response for first request")

	// cache hit
	respCached, err := CachingInterceptor(ctx, req1, info, mockHandler)
	assert.NoError(t, err, "Error in cached request")
	assert.Equal(t, "response-request1", respCached, "Unexpected response for cached request")

	// Verify handler not called again
	assert.Equal(t, resp, respCached, "Cached response did not match original response")

	// Different request - cache miss
	req2 := "request2"
	resp2, err := CachingInterceptor(ctx, req2, info, mockHandler)
	assert.NoError(t, err, "Error in second request")
	assert.Equal(t, "response-request2", resp2, "Unexpected response for second request")

	// Add a third request
	req3 := "request3"
	resp3, err := CachingInterceptor(ctx, req3, info, mockHandler)
	assert.NoError(t, err, "Error in third request")
	assert.Equal(t, "response-request3", resp3, "Unexpected response for third request")

	// The first request should have been evicted due to cache size.
	respEvicted, ok := cache.Get(generateCacheKey(info.FullMethod, req1))
	assert.False(t, ok, "Expected first request to be evicted from cache")
	assert.Nil(t, respEvicted, "Evicted cache entry should be nil")
}
