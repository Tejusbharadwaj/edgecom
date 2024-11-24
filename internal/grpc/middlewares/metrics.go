package middleware

import (
	"context"
	"path"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

func NewMetricsInterceptor(
	requests *prometheus.CounterVec,
	latency *prometheus.HistogramVec,
) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start).Seconds()
		method := path.Base(info.FullMethod)

		requests.WithLabelValues(method).Inc()
		latency.WithLabelValues(method).Observe(duration)

		return resp, err
	}
}
