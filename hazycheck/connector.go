package hazycheck

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

const clientTimeout = time.Second * 5

// Preconfigured http client, for example, uses retries, uses predefined User-Agent headers,
// to avoid filtering the checksystem traffic, predefined timeouts to avoid checker stucks.
// More over, this http client will be profiled to gather metrics from it.

type Connector interface {
	// GetHTTPClient returns preconfigured http client.
	GetHTTPClient() (*http.Client, error)
}

type connectorImpl struct {
	t http.RoundTripper
}

func NewConnector(svc string) *connectorImpl {
	// client will be used only for one host (in theory)
	// thats why conns settings have the same numbert.
	var t http.RoundTripper = &http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: false,

		// TODO: move values to configuration
		MaxIdleConns:        30,
		MaxIdleConnsPerHost: 30,
		MaxConnsPerHost:     30,
		IdleConnTimeout:     time.Minute * 5,
	}
	t = newInstrumentedRoundTripperFor(t, svc)

	return &connectorImpl{t: t}
}

// GetHTTPClient returns preconfigured http client. Preconfigured client will
// have the default timeout (5s) on requests to avoid dangling checks. Also,
// client's transport will be instrumented, that will allow us to get some metrics
// about service/checker state
func (c *connectorImpl) GetHTTPClient() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic("unreachable: cannot create cookie jar")
	}

	// it's OK to create new http clients. Connection pool is controlled by transport.
	return &http.Client{
		Transport: c.t,
		Jar:       jar,

		// TODO: move values to configuration
		Timeout: clientTimeout,
	}, nil
}

type instrumentedRoundTripper struct {
	inner http.RoundTripper
	svc   string
}

func newInstrumentedRoundTripperFor(inner http.RoundTripper, svc string) *instrumentedRoundTripper {
	return &instrumentedRoundTripper{
		inner: inner,
		svc:   svc,
	}
}

func (t *instrumentedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	target := req.URL.Host
	method := req.Method
	reqStart := time.Now()

	rsp, err := t.inner.RoundTrip(req)

	statusCode := rsp.StatusCode

	if err != nil {
		errCnt := metrics.GetOrCreateCounter(
			fmt.Sprintf(
				`govnilo_http_request_errors{method=%q, target=%q, service=%q}`,
				method,
				target,
				t.svc,
			),
		)
		errCnt.Inc()

		return nil, err
	}

	reqMetrics := metrics.GetOrCreateHistogram(
		fmt.Sprintf(
			`govnilo_http_requests_duration{method=%q, target=%q, service=%q, status=%q}`,
			method,
			target,
			t.svc,
			strconv.Itoa(statusCode),
		),
	)
	reqMetrics.UpdateDuration(reqStart)

	return rsp, nil
}
