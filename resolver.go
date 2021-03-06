package khs

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/resolver"
)

const (
	name        = "khs"
	updateEvery = time.Minute
)

// Register khs with gRPC.
func init() {
	resolver.Register(&khsBuilder{})
}

// khsBuilder is the Builder for a KHS Resolver. It's immutable, so is safe to be registered multiple times.
type khsBuilder struct{}

// Build parses the target for the service host and the endpoint port, returning an error if these can not be parsed.
// Should this succeed, it initialises a khsResolver, calls the first resolve and if this completes, it's returned.
func (kb *khsBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	strs := strings.Split(target.Endpoint, ":")

	if len(strs) > 2 || len(strs) <= 0 {
		return nil, errors.New(fmt.Sprintf("couldn't parse given target endpoint: %s", target.Endpoint))
	}

	res := &khsResolver{
		cc:          cc,
		serviceHost: strs[0],
		quitC:       make(chan struct{}),
	}

	if len(strs) == 2 {
		port, err := strconv.Atoi(strs[1])

		if err != nil {
			return nil, errors.New(fmt.Sprintf("couldn't parse given port: %s", strs[1]))
		}

		res.endpointPort = port
	}

	err := res.resolve()

	if err != nil {
		return nil, err
	}

	go res.periodicUpdate()

	return res, nil
}

// Scheme returns `khs`.
func (kb *khsBuilder) Scheme() string {
	return name
}

// khsResolver is the resolver for Kubernetes Headless Services. When called, it looks up all the A Records for the given
// Host, passing them to a ClientConn as Backends.
type khsResolver struct {
	cc resolver.ClientConn

	serviceHost  string
	endpointPort int

	quitC chan struct{}
}

// resolve calls the ClientConn NewAddress callback with all the IPs returned from a standard DNS lookup to the serviceHost,
// affixing kr.endpointPort to each of them.
func (kr *khsResolver) resolve() error {
	ips, err := net.LookupIP(kr.serviceHost)

	if err != nil {
		return err
	}

	addrs := make([]resolver.Address, len(ips))

	for i, ip := range ips {
		addr := ip.String()

		if kr.endpointPort != 0 {
			addr = fmt.Sprintf("%s:%d", addr, kr.endpointPort)
		}

		addrs[i] = resolver.Address{
			Addr:       addr,
			ServerName: kr.serviceHost,
		}
	}

	// NOTE: Use of the built-in Round Robin Balancer (google.golang.org/grpc/balancer/roundrobin) is now set via
	// ServiceConfig JSON instead of the depreciated grpc.WithBalancerName(roundrobin.Name), previously a client DialOption.
	// However, the gRPC Service Config docs (https://github.com/grpc/grpc/blob/master/doc/service_config.md) suggest
	// loadBalancingPolicy is also being deprecated with no clear alternative.
	//
	// grpc/service_config.go currently supports a 'loadBalancingConfig' field, however it looks likely to change, so for
	// now stick to the existing JSON definition.
	kr.cc.UpdateState(resolver.State{
		Addresses:     addrs,
		ServiceConfig: kr.cc.ParseServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
	})

	return nil
}

// periodicUpdate periodically calls resolve to ensure kr.cc contains an recent list of the service endpoints.
func (kr *khsResolver) periodicUpdate() {
	t := time.NewTicker(updateEvery)
	for {
		select {
		case <-t.C:
			err := kr.resolve()

			if err != nil {
				log.Printf("[khs - resolver.go] error resolving: %v", err)
			}
		case <-kr.quitC:
			return
		}
	}
}

// Resolve now runs an internal resolve, updating khs.cc with the current list of endpoints.
func (kr *khsResolver) ResolveNow(option resolver.ResolveNowOptions) {
	err := kr.resolve()

	if err != nil {
		log.Printf("[khs - resolver.go] error resolving: %v", err)
	}
}

func (kr *khsResolver) Close() {
	kr.quitC <- struct{}{}
}
