package khs

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"google.golang.org/grpc/resolver"
)

const name = "khs"

// Register khs with gRPC.
func init() {
	resolver.Register(&khsBuilder{})
}

// khsBuilder is the Builder for a KHS Resolver. It's immutable, so is safe to be registered multiple times.
type khsBuilder struct{}

// Build parses the target for the service host and the endpoint port, returning an error if these can not be parsed.
// Should this succeed, it initialises a khsResolver, calls the first resolve and if this completes, it's returned.
func (kb *khsBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	strs := strings.Split(target.Endpoint, ":")

	if len(strs) > 2 || len(strs) <= 0 {
		return nil, errors.New(fmt.Sprintf("couldn't parse given target endpoint: %s", target.Endpoint))
	}

	res := &khsResolver{
		cc:          cc,
		serviceHost: strs[0],
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
			Addr: addr,
			Type: resolver.Backend,
		}
	}

	kr.cc.NewAddress(addrs)

	return nil
}

// Resolve now runs an internal resolve, updating khs.cc with the current list of endpoints.
func (kr *khsResolver) ResolveNow(option resolver.ResolveNowOption) {
	err := kr.resolve()

	if err != nil {
		log.Printf("[khs - resolver.go] error resolving: %v", err)
	}
}

func (kr *khsResolver) Close() {

}
