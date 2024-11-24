package middleware

import (
	"context"
	"encoding/json"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"google.golang.org/grpc"
)

type Cache struct {
	cache *lru.Cache
}

// This in-memory cache is used for simplicity purpose. It can be replaced with Redis.
// golang-lru Automatically evicts the least recently accessed items, ensuring efficient memory usage.

func NewCache(size int) (*Cache, error) {
	c, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &Cache{cache: c}, nil
}

func (c *Cache) InterceptorFunc() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		key := generateCacheKey(info.FullMethod, req)

		if cachedResp, ok := c.cache.Get(key); ok {
			return cachedResp, nil
		}

		resp, err := handler(ctx, req)
		if err != nil {
			return nil, err
		}

		c.cache.Add(key, resp)
		return resp, nil
	}
}

func generateCacheKey(method string, req interface{}) string {
	reqBytes, _ := json.Marshal(req)
	return fmt.Sprintf("%s:%s", method, string(reqBytes))
}
