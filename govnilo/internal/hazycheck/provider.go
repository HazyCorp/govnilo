package hazycheck

import (
	"net/http"

	"google.golang.org/grpc"
)

// Preconfigured http client, for example, uses retries, uses predefined User-Agent headers,
// to avoid filtering the checksystem traffic, predefined timeouts to avoid checker stucks.
// More over, this http client will be profiled to gather metrics from it.

type Provider interface {
	// GetHTTPClient returns preconfigured http client.
	GetHTTPClient() (*http.Client, error)
	GRPCConn(
		target string,
		opts ...grpc.DialOption,
	) (*grpc.ClientConn, error)
}

type Dummy struct{}

func NewDummyProvider() *Dummy {
	return &Dummy{}
}

// GetHTTPClient returns preconfigured http client.
func (d *Dummy) GetHTTPClient() (*http.Client, error) {
	return http.DefaultClient, nil
}

func (d *Dummy) GRPCConn(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return grpc.Dial(target, opts...)
}
