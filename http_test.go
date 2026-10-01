package fiberfx_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-core-fx/fiberfx"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestFiberConfig(t *testing.T) {
	defaultApp := fiberfx.New(fiberfx.Config{}, fiberfx.Options{}, zap.NewNop())
	if got := defaultApp.Config().BodyLimit; got != fiber.DefaultBodyLimit {
		t.Fatalf("default body limit: got %d, want %d", got, fiber.DefaultBodyLimit)
	}

	const limit = 30 * 1024 * 1024
	views := &testViews{}
	options := fiberfx.Options{}
	options.WithFiberConfig(func(cfg *fiber.Config) {
		cfg.BodyLimit = limit
		cfg.ProxyHeader = "X-Real-IP"
		cfg.GETOnly = true
		cfg.Views = views
		cfg.ErrorHandler = func(c *fiber.Ctx, _ error) error {
			return c.SendStatus(fiber.StatusConflict)
		}
	})
	app := fiberfx.New(fiberfx.Config{ProxyHeader: "X-Forwarded-For"}, options, zap.NewNop())
	if got := app.Config().BodyLimit; got != limit {
		t.Fatalf("body limit: got %d, want %d", got, limit)
	}
	if got := app.Config().ProxyHeader; got != "X-Real-IP" {
		t.Fatalf("proxy header: got %q, want X-Real-IP", got)
	}
	if !app.Config().GETOnly || app.Config().Views != views {
		t.Fatal("Fiber options not applied")
	}

	app.Get("/failure", func(_ *fiber.Ctx) error { return fiber.ErrNotImplemented })
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/failure", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	if response.StatusCode != fiber.StatusConflict {
		t.Fatalf("custom error handler: got %d, want %d", response.StatusCode, fiber.StatusConflict)
	}
}

type testViews struct{}

func (*testViews) Load() error { return nil }

func (*testViews) Render(_ io.Writer, _ string, _ any, _ ...string) error { return nil }

func TestRequestLogFields(t *testing.T) {
	for _, test := range []struct {
		name     string
		options  func(*fiberfx.Options)
		wantBody bool
	}{
		{name: "default", options: func(*fiberfx.Options) {}, wantBody: true},
		{name: "empty", options: func(o *fiberfx.Options) {
			o.WithRequestLogFields()
		}, wantBody: true},
		{name: "empty slice", options: func(o *fiberfx.Options) {
			fields := []string{}
			o.WithRequestLogFields(fields...)
		}, wantBody: true},
		{name: "without body", options: func(o *fiberfx.Options) {
			o.WithRequestLogFields("requestId", "status", "error")
		}, wantBody: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			options := fiberfx.Options{}
			test.options(&options)
			app := fiberfx.New(fiberfx.Config{}, options, zap.New(core))
			app.Post("/upload", func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusNotImplemented)
			})

			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://example.com/upload",
				strings.NewReader("test upload"))
			if err != nil {
				t.Fatal(err)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = response.Body.Close() })
			if response.StatusCode != fiber.StatusNotImplemented {
				t.Fatalf("status: got %d, want %d", response.StatusCode, fiber.StatusNotImplemented)
			}
			if logs.Len() != 1 {
				t.Fatalf("log entries: got %d, want 1", logs.Len())
			}
			fields := logs.All()[0].ContextMap()
			_, hasBody := fields["body"]
			if hasBody != test.wantBody {
				t.Fatalf("body logged: got %v, want %v", hasBody, test.wantBody)
			}
			if _, ok := fields["status"]; !ok {
				t.Fatal("status field is missing")
			}
		})
	}
}
