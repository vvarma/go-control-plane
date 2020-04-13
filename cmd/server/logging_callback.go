package main

import (
	"context"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/log"
)

type loggingCallback struct {
}

func (l loggingCallback) OnStreamOpen(ctx context.Context, streamId int64, url string) error {
	log.Infof("Opening stream id: %d url: %s ads: %t", streamId, url, len(url) == 0)
	return nil
}

func (l loggingCallback) OnStreamClosed(streamId int64) {
	log.Infof("Closing stream id: %d", streamId)
}

func (l loggingCallback) OnStreamRequest(streamId int64, req *discoverygrpc.DiscoveryRequest) error {
	log.Infof("Discovery Request on stream id: %d Node: %s Type: %s", streamId, req.Node.Id, req.TypeUrl)
	return nil
}

func (l loggingCallback) OnStreamResponse(streamId int64, req *discoverygrpc.DiscoveryRequest, resp *discoverygrpc.DiscoveryResponse) {
	log.Infof("Discovery Response on stream id: %d Node: %s Type: %s", streamId, req.Node.Id, req.TypeUrl)
}

func (l loggingCallback) OnFetchRequest(ctx context.Context, req *discoverygrpc.DiscoveryRequest) error {
	log.Infof("Fetch Request on stream id: %d Node: %s Type: %s", req.Node, req.TypeUrl)
	return nil
}

func (l loggingCallback) OnFetchResponse(*discoverygrpc.DiscoveryRequest, *discoverygrpc.DiscoveryResponse) {
}
