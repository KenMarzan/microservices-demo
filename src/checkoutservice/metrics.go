// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	// gRPC request metrics
	grpcRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"service", "method", "status_code"},
	)

	grpcRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "gRPC request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method"},
	)

	grpcErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_errors_total",
			Help: "Total number of gRPC errors",
		},
		[]string{"service", "method", "error_code"},
	)
)

// prometheusUnaryInterceptor is a gRPC unary interceptor that collects Prometheus metrics
func prometheusUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start).Seconds()
		method := info.FullMethod
		
		statusCode := "OK"
		if err != nil {
			st, _ := status.FromError(err)
			statusCode = st.Code().String()
			grpcErrorsTotal.WithLabelValues("checkoutservice", method, statusCode).Inc()
		}

		grpcRequestsTotal.WithLabelValues("checkoutservice", method, statusCode).Inc()
		grpcRequestDuration.WithLabelValues("checkoutservice", method).Observe(duration)

		return resp, err
	}
}

// startMetricsServer starts an HTTP server to expose Prometheus metrics
func startMetricsServer(port string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	
	addr := fmt.Sprintf(":%s", port)
	log.Infof("Starting metrics server on %s", addr)
	
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Errorf("Failed to start metrics server: %v", err)
		}
	}()
}
