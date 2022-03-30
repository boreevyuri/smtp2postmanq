package healthcheck

import (
	"context"
	"errors"
	"net/http"
	"runtime"
	"time"

	"github.com/alexliesenfeld/health"
)

type Handler struct {
	handler      http.HandlerFunc
	checks       map[string]health.CheckerOption
	routineCount int
	dnsTimeout   time.Duration
	dnsCheck     map[string]string
}

type IHealthHandler interface {
	Init()
	AddCheck(name string, check health.CheckerOption)
	GetEndpoint() http.HandlerFunc
	SetMaxRoutineCount(cnt int)
}

func Provide() IHealthHandler {
	return &Handler{
		checks:       make(map[string]health.CheckerOption),
		routineCount: 100,
		dnsTimeout:   50 * time.Millisecond,
		dnsCheck:     make(map[string]string),
	}
}

func (h *Handler) Init() {
	checks := []health.CheckerOption{
		// Set the time-to-live for our cache to 1 second (default).
		health.WithCacheDuration(1 * time.Second),

		// Configure a global timeout that will be applied to all checks.
		health.WithTimeout(10 * time.Second),

		health.WithCheck(health.Check{
			Name:  "goroutine-threshold",
			Check: h.routineChecker,
		}),
	}

	h.handler = health.NewHandler(health.NewChecker(
		checks...,
	))
}

func (h *Handler) AddCheck(name string, check health.CheckerOption) {
	h.checks[name] = check
}

func (h *Handler) GetEndpoint() http.HandlerFunc {
	return h.handler
}

func (h *Handler) SetMaxRoutineCount(cnt int) {
	h.routineCount = cnt
}

func (h *Handler) routineChecker(_ context.Context) error {
	if runtime.NumGoroutine() > h.routineCount {
		return errors.New("too much routines")
	}

	return nil
}
