package adminklient

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/HazyCorp/govnilo/pkg/common/checkersettings"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
)

const (
	defaultRefreshInterval = time.Second * 5
	defaultTimeout         = time.Second * 3
)

type AsyncClientConfig struct {
	RefreshInterval time.Duration `json:"refresh_interval" yaml:"refresh_interval"`
	Timeout         time.Duration `json:"timeout" yaml:"timeout"`
}

type AsyncClient struct {
	c         AsyncClientConfig
	inner     Client
	lastState atomic.Pointer[checkersettings.Settings]
	l         *slog.Logger
}

func NewAsync(inner Client, conf AsyncClientConfig, opts ...ClientOpt) *AsyncClient {
	var o clientOptions
	for _, opt := range opts {
		opt.apply(&o)
	}

	if o.Logger == nil {
		o.Logger = hzlog.NopLogger()
	}
	o.Logger = o.Logger.With(slog.String("component", "infra:async_adminka_client"))

	if conf.RefreshInterval == 0 {
		conf.RefreshInterval = defaultRefreshInterval
	}
	if conf.Timeout == 0 {
		conf.Timeout = defaultTimeout
	}

	c := &AsyncClient{
		c:     conf,
		inner: inner,
		l:     o.Logger,
	}
	go c.run()

	return c
}

func (c *AsyncClient) run() {
	sync := func() {
		c.l.Debug("trying to update the settings")

		tCtx, tCancel := context.WithTimeout(context.Background(), c.c.Timeout)
		current, err := c.inner.GetConfig(tCtx)
		tCancel()
		if err != nil {
			c.l.Warn("cannot update settings", hzlog.Error(err))
		}

		c.l.Debug("successfully updated the settings")
		c.lastState.Store(current)
	}

	// sync before the first tick
	sync()

	ticker := time.NewTicker(c.c.RefreshInterval)
	for range ticker.C {
		sync()
	}
}

func (c *AsyncClient) GetConfig(ctx context.Context) (*checkersettings.Settings, error) {
	ptr := c.lastState.Load()
	if ptr == nil {
		return nil, errors.New("cannot get config asyncroniously, state is not known yet")
	}

	return ptr, nil
}
