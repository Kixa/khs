# khs

khs is a resolver for Headless Services in Kubernetes used for (EXPERIMENTAL) Load Balancing gRPC in the client (at L4) in go.


## Usage

Add `_ "github.com/kixa/khs"` to your imports and use the `khs` scheme to dial your `Headless Service`.

For Example (with the built-in `roundrobin` balancer):

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
	conn, err := grpc.Dial("khs:///example.default<:optional_port>", grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name))

	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
    
    // ... Initialise service and call remote methods...
}
```