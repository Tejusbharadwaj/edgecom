package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Mock request for testing
type mockRequest struct {
	Start       *timestamppb.Timestamp
	End         *timestamppb.Timestamp
	Window      string
	Aggregation string
}

func TestCache(t *testing.T) {
	t.Run("cache operations", func(t *testing.T) {
		// Initialize cache
		cache, err := NewCache(2)
		require.NoError(t, err)

		// Setup test data
		now := time.Now()
		req := &mockRequest{
			Start:       timestamppb.New(now.Add(-time.Hour)),
			End:         timestamppb.New(now),
			Window:      "1h",
			Aggregation: "AVG",
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/Method",
		}

		// Mock handler that counts calls
		callCount := 0
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			callCount++
			return "response", nil
		}

		interceptor := cache.InterceptorFunc()

		// First call - should miss cache
		resp1, err := interceptor(context.Background(), req, info, handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp1)
		assert.Equal(t, 1, callCount)

		// Second call with same request - should hit cache
		resp2, err := interceptor(context.Background(), req, info, handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp2)
		assert.Equal(t, 1, callCount, "Handler should not be called on cache hit")

		// Different request - should miss cache
		req2 := &mockRequest{
			Start:       timestamppb.New(now.Add(-2 * time.Hour)),
			End:         timestamppb.New(now),
			Window:      "1h",
			Aggregation: "MAX",
		}
		resp3, err := interceptor(context.Background(), req2, info, handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp3)
		assert.Equal(t, 2, callCount)
	})

	t.Run("cache eviction", func(t *testing.T) {
		// Initialize cache with size 1
		cache, err := NewCache(1)
		require.NoError(t, err)

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/Method",
		}

		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "response", nil
		}

		interceptor := cache.InterceptorFunc()

		// First request
		req1 := &mockRequest{Window: "1h"}
		_, err = interceptor(context.Background(), req1, info, handler)
		assert.NoError(t, err)

		// Second request - should evict first
		req2 := &mockRequest{Window: "2h"}
		_, err = interceptor(context.Background(), req2, info, handler)
		assert.NoError(t, err)

		// Verify first request was evicted
		key := generateCacheKey(info.FullMethod, req1)
		_, ok := cache.cache.Get(key)
		assert.False(t, ok, "First request should have been evicted")
	})

	t.Run("handler error", func(t *testing.T) {
		cache, err := NewCache(1)
		require.NoError(t, err)

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/Method",
		}

		// Create a test request
		req := &mockRequest{Window: "1h"}

		// Handler that returns error
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, assert.AnError
		}

		interceptor := cache.InterceptorFunc()

		// Verify error is propagated and not cached
		resp, err := interceptor(context.Background(), req, info, handler)
		assert.Error(t, err)
		assert.Nil(t, resp)

		// Verify the error response wasn't cached
		key := generateCacheKey(info.FullMethod, req)
		_, ok := cache.cache.Get(key)
		assert.False(t, ok, "Error responses should not be cached")
	})
}
