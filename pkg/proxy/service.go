package proxy

import (
	"context"

	grpcProxyV2 "github.com/hellofresh/kangal/pkg/proxy/rpc/pb/grpc/proxy/v2"
)

type implLoadTestServiceServer struct {
	grpcProxyV2.UnsafeLoadTestServiceServer
}

// NewLoadTestServiceServer instantiates new LoadTestServiceServer implementation
func NewLoadTestServiceServer() grpcProxyV2.LoadTestServiceServer {
	return &implLoadTestServiceServer{}
}

// List searches and returns load tests by given filters
func (s *implLoadTestServiceServer) List(context.Context, *grpcProxyV2.ListRequest) (*grpcProxyV2.ListResponse, error) {
	return new(grpcProxyV2.ListResponse), nil
}

// Get returns load test by given name
func (s *implLoadTestServiceServer) Get(context.Context, *grpcProxyV2.GetRequest) (*grpcProxyV2.GetResponse, error) {
	return new(grpcProxyV2.GetResponse), nil
}
