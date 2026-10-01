package fiberfx

import (
	"slices"

	"github.com/gofiber/fiber/v2"
)

type Config struct {
	Address     string
	ProxyHeader string
	Proxies     []string
}

type Options struct {
	configureFiber   func(*fiber.Config)
	requestLogFields []string
	withMetrics      bool
}

func (o *Options) WithMetrics() *Options {
	o.withMetrics = true
	return o
}

// WithFiberConfig applies changes after the default Fiber config is assembled.
func (o *Options) WithFiberConfig(configure func(*fiber.Config)) *Options {
	o.configureFiber = configure
	return o
}

// WithRequestLogFields replaces the HTTP logger fields; an empty list uses the defaults.
func (o *Options) WithRequestLogFields(fields ...string) *Options {
	o.requestLogFields = slices.Clone(fields)
	return o
}
