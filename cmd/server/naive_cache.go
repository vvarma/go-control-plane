package main

import (
	"context"
	envoyapiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoyapiv2route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoyconfigclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoyconfigcorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyconfigendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	httpConnectionManagerV2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoyconfiglistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/log"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"time"
)

type ns struct {
	count               int
	clusterRefreshCount int
}

func NaiveCache() cache.Cache {
	return &ns{0, 0}
}

func (n *ns) CreateWatch(req cache.Request) (value chan cache.Response, cancel func()) {
	log.Infof("Watch request - node: %s type: %s resource: %v ", req.Node.Id, req.TypeUrl, req.ResourceNames)
	ctx, cancel := context.WithCancel(context.Background())
	value = make(chan cache.Response)
	go func() {
		tc := time.Tick(time.Second * 120)
		for {
			select {
			case <-ctx.Done():
				break
			case <-tc:
				up := n.generateUpdate(req)
				if up != nil {
					value <- *up
				}
				//default:
			}
		}
	}()
	return
}

/*
   static_resources:
     listeners:
     - name: listener_0
       address:
         socket_address:
           address: 127.0.0.1
           port_value: 4891
       filter_chains:
       - filters:
         - name: envoy.http_connection_manager
           typed_config:
             "@type": type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
             stat_prefix: ingress_http
             codec_type: HTTP2
             route_config:
               name: local_route
               virtual_hosts:
               - name: local_service
                 domains: ["*"]
                 routes:
                 - match:
                     prefix: "/"
                   route:
                     cluster: voyager_server_service
             http_filters:
             - name: envoy.router
     clusters:
     - name: voyager_server_service
       connect_timeout: 0.25s
       http2_protocol_options: {}
       type: STRICT_DNS
       lb_policy: ROUND_ROBIN
       load_assignment:
         cluster_name: local_service
         endpoints:
         - lb_endpoints:
           - endpoint:
               address:
                 socket_address:
                   address: voyager-sever
                   port_value: 4891
*/

func (n *ns) generateUpdate(req cache.Request) *cache.Response {
	switch req.TypeUrl {
	case resource.EndpointType:
		return nil
	case resource.ClusterType:
		serverAddresses := []string{
			"voyager-server",
			"voyager-server-fallback",
		}
		address := serverAddresses[n.clusterRefreshCount%len(serverAddresses)]
		log.Infof("Cluster Refresh count is %d adress selected %s ", n.clusterRefreshCount, address)
		var clusters []types.Resource
		clusters = append(clusters, &envoyconfigclusterv3.Cluster{
			Name: "voyager_server_service",
			ConnectTimeout: &duration.Duration{
				Nanos: 2.5e8,
			},
			Http2ProtocolOptions: &envoyconfigcorev3.Http2ProtocolOptions{},
			ClusterDiscoveryType: &envoyconfigclusterv3.Cluster_Type{Type: envoyconfigclusterv3.Cluster_STRICT_DNS},
			LoadAssignment: &envoyconfigendpointv3.ClusterLoadAssignment{
				ClusterName: "local_service",
				Endpoints: []*envoyconfigendpointv3.LocalityLbEndpoints{
					{
						LbEndpoints: []*envoyconfigendpointv3.LbEndpoint{
							{
								HostIdentifier: &envoyconfigendpointv3.LbEndpoint_Endpoint{
									Endpoint: &envoyconfigendpointv3.Endpoint{
										Address: &envoyconfigcorev3.Address{
											Address: &envoyconfigcorev3.Address_SocketAddress{
												SocketAddress: &envoyconfigcorev3.SocketAddress{
													Protocol: envoyconfigcorev3.SocketAddress_TCP,
													Address:  address,
													PortSpecifier: &envoyconfigcorev3.SocketAddress_PortValue{
														PortValue: 4891},
												}},
										},
									},
								},
							},
						},
					},
				},
			},
		})

		resp := cache.Response{
			Request:           req,
			ResourceMarshaled: false,
			Resources:         clusters,
		}
		n.clusterRefreshCount += 1
		return &resp
	case resource.ListenerType:
		var listeners []types.Resource
		httpFilter := httpConnectionManagerV2.HttpConnectionManager{
			CodecType:  httpConnectionManagerV2.HttpConnectionManager_HTTP2,
			StatPrefix: "voyager_server_egress",
			RouteSpecifier: &httpConnectionManagerV2.HttpConnectionManager_RouteConfig{
				RouteConfig: &envoyapiv2.RouteConfiguration{
					Name: "local_route",
					VirtualHosts: []*envoyapiv2route.VirtualHost{
						{
							Name: "local_service",
							Domains: []string{
								"*",
							},
							Routes: []*envoyapiv2route.Route{
								{
									Match: &envoyapiv2route.RouteMatch{
										PathSpecifier: &envoyapiv2route.RouteMatch_Prefix{Prefix: "/"},
									},
									Action: &envoyapiv2route.Route_Route{Route: &envoyapiv2route.RouteAction{
										ClusterSpecifier: &envoyapiv2route.RouteAction_Cluster{Cluster: "voyager_server_service"},
									}},
								},
							},
						},
					},
				},
			},
			HttpFilters: []*httpConnectionManagerV2.HttpFilter{
				{Name: "envoy.router"},
			},
		}
		httpFilterAny, er := ptypes.MarshalAny(&httpFilter)
		if er != nil {
			log.Error(er)
			return nil
		}
		httpFilterConfig := envoyconfiglistenerv3.Filter{
			Name:       "envoy.http_connection_manager",
			ConfigType: &envoyconfiglistenerv3.Filter_TypedConfig{TypedConfig: httpFilterAny},
		}
		listeners = append(listeners, &envoyconfiglistenerv3.Listener{
			Name: "listener_voyager_sever",
			Address: &envoyconfigcorev3.Address{
				Address: &envoyconfigcorev3.Address_SocketAddress{
					SocketAddress: &envoyconfigcorev3.SocketAddress{
						Protocol:     envoyconfigcorev3.SocketAddress_TCP,
						Address:      "127.0.0.1",
						ResolverName: "",
						Ipv4Compat:   false,
						PortSpecifier: &envoyconfigcorev3.SocketAddress_PortValue{
							PortValue: 4891,
						},
					},
				},
			},
			FilterChains: []*envoyconfiglistenerv3.FilterChain{
				{
					Filters: []*envoyconfiglistenerv3.Filter{
						&httpFilterConfig,
					},
				},
			},
		})
		resp := cache.Response{
			Request:           req,
			ResourceMarshaled: false,
			Resources:         listeners,
		}
		return &resp
	default:
		return nil
	}
}

func (n *ns) Fetch(ctx context.Context, req cache.Request) (*cache.Response, error) {
	log.Infof("Fetch request - node: %s type: %s resource: %v ", req.Node, req.TypeUrl, req.ResourceNames)
	up := n.generateUpdate(req)
	return up, nil
}
