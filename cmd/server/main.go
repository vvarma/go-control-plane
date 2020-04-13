package main

import (
	"context"
	"flag"
	"fmt"
	accessloggrpc "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	testv3 "github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	grpcMaxConcurrentStreams = 1000000
)

var (
	port    uint
	alsPort uint
)

func init() {
	flag.UintVar(&port, "port", 18000, "Management server port")
	flag.UintVar(&alsPort, "als", 18090, "Accesslog server port")
}

func runADS(ctx context.Context, configv3 cachev3.Cache) {
	server := serverv3.NewServer(context.Background(), configv3, loggingCallback{})

	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems.
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
	grpcServer := grpc.NewServer(grpcOptions...)
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Println("Starting ADS grpc server on ", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Println(err)
		}
	}()
	<-ctx.Done()
	grpcServer.GracefulStop()

}
func runALS(ctx context.Context) {
	als := &testv3.AccessLogService{}
	grpcServer := grpc.NewServer()
	accessloggrpc.RegisterAccessLogServiceServer(grpcServer, als)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", alsPort))
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Println("Starting ALS grpc server on ", alsPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Println(err)
		}
	}()
	go func() {
		for {
			tc := time.Tick(time.Second)
			select {
			case <-ctx.Done():
				break
			case <-tc:
				printer := func(s string) {
					log.Println(s)
				}
				als.Dump(printer)
			}
		}
	}()
	<-ctx.Done()
	grpcServer.GracefulStop()
}

func main() {
	flag.Parse()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	cache := NaiveCache()

	go runADS(ctx, cache)
	go runALS(ctx)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	cancel()
}
