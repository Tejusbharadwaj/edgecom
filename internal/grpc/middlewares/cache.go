package middleware

// This in-memory cache is used for simplicity purpose. It can be replaced with Redis.
// golang-lru Automatically evicts the least recently accessed items, ensuring efficient memory usage.

import (
	"context"
	"encoding/json"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"google.golang.org/grpc"
)

var cache *lru.Cache

// InitializeCache sets up an in-memory LRU cache.
func InitializeCache(size int) error {
	var err error
	cache, err = lru.New(size) // Create an LRU cache with the specified size
	return err
}

// CachingInterceptor is a gRPC middleware for caching responses in memory.
func CachingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// Generate a cache key based on the gRPC method and request.
	key := generateCacheKey(info.FullMethod, req)

	// Check the in-memory cache for a response.
	if cachedResp, ok := cache.Get(key); ok {
		return cachedResp, nil // Cache hit: return the cached response.
	}

	// Cache miss: Proceed with the handler.
	resp, err := handler(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store the response in the cache.
	cache.Add(key, resp)
	return resp, nil
}

// generateCacheKey generates a cache key based on the gRPC method and request.
func generateCacheKey(method string, req interface{}) string {
	// Serialize the request to create a unique cache key.
	reqBytes, _ := json.Marshal(req) // Ignore errors for simplicity.
	return fmt.Sprintf("%s:%s", method, string(reqBytes))
}
