# khs

khs is a gRPC resolver for [Headless Services](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services) in Kubernetes used for Round Robin Load Balancing (at L4) in Go.

As per [Load Balancing in gRPC](https://github.com/grpc/grpc/blob/master/doc/load-balancing.md), khs enables a Balancing-Aware Client, which -as stated- has many drawbacks. However, if all (or most) of your gRPC clients are written in Go, you don't want to run Istio or write a Load Balancing service and you don't need a more complicated balancing strategy than Round Robin, this makes things very simple. 

Since this project follows the grpc-go version (i.e. If you're using 1.26.X, you should use khs 1.26.X) and the resolver/balancer API is classed as EXPERIMENTAL, it does not follow the Go compatibility promise between large releases.

## Usage

Add `_ "github.com/kixa/khs"` to your imports and use the `khs` scheme to dial your `Headless Service`.

For Example:

```
package main

import (
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"

	_ "github.com/kixa/khs"
)

func main() {
	conn, err := grpc.Dial("khs:///example.default<:optional_port>", grpc.WithInsecure())

	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
    
    // ... Initialise service and call remote methods...
}
```