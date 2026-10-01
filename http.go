package fiberfx

import (
	"strings"

	"github.com/go-core-fx/fiberfx/prometheus"
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"go.uber.org/zap"
)

func New(config Config, option Options, logger *zap.Logger) *fiber.App {
	fields := []string{"requestId", "latency", "status", "method", "url", "ip", "ua", "body", "error"}
	if len(option.requestLogFields) > 0 {
		fields = option.requestLogFields
	}

	fiberConfig := fiber.Config{
		DisableStartupMessage:   true,
		EnableIPValidation:      true,
		EnableTrustedProxyCheck: len(config.Proxies) > 0,
		ProxyHeader:             config.ProxyHeader,
		TrustedProxies:          config.Proxies,
		UnescapePath:            true,
	}
	if option.configureFiber != nil {
		option.configureFiber(&fiberConfig)
	}

	app := fiber.New(fiberConfig)
	app.Use(requestid.New())
	app.Use(fiberzap.New(fiberzap.Config{
		Next: func(c *fiber.Ctx) bool {
			p := c.Path()
			// Normalize trailing slash
			for len(p) > 1 && p[len(p)-1] == '/' {
				p = p[:len(p)-1]
			}

			return p == "/health" || p == "/metrics" ||
				strings.HasPrefix(p, "/health/") || strings.HasPrefix(p, "/metrics/")
		},
		SkipBody: func(c *fiber.Ctx) bool {
			return c.Response().StatusCode() < fiber.StatusBadRequest
		},
		Logger: logger,
		Fields: fields,
	}))
	app.Use(recover.New())

	if option.withMetrics {
		prometheus.Register(app)
	}

	return app
}
