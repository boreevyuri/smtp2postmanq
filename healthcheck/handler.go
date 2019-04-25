package healthcheck

import (
	"net/http"
	"time"

	"github.com/Halfi/healthcheck"
	"github.com/jinzhu/gorm"
)

type Handler struct {
	handler      healthcheck.Handler
	checks       map[string]healthcheck.Check
	routineCount int
	dnsTimeout   time.Duration
	dnsCheck     map[string]string
	db           *gorm.DB
	dbTimeout    time.Duration
}

type IHealthHandler interface {
	Init()
	AddCheck(name string, check healthcheck.Check)
	GetLiveEndpoint() func(http.ResponseWriter, *http.Request)
	GetReadyEndpoint() func(http.ResponseWriter, *http.Request)
	SetMaxRoutineCount(cnt int)
	AddDnsCheck(name string, host string)
	AddDBCheck(db *gorm.DB)
}

func ProvideHandler() IHealthHandler {
	return &Handler{
		handler:      healthcheck.NewHandler(),
		checks:       make(map[string]healthcheck.Check),
		routineCount: 100,
		dnsTimeout:   50 * time.Millisecond,
		dnsCheck:     make(map[string]string),
		dbTimeout:    1 * time.Second,
	}
}

func (h *Handler) Init() {
	h.handler.AddLivenessCheck("goroutine-threshold", healthcheck.GoroutineCountCheck(h.routineCount))

	for name, host := range h.dnsCheck {
		h.handler.AddLivenessCheck(name, healthcheck.DNSResolveCheck(host, h.dnsTimeout))
	}

	if h.db != nil {
		db := h.db.DB()
		if db != nil {
			h.handler.AddLivenessCheck("database", healthcheck.DatabasePingCheck(db, h.dbTimeout))
		}
	}

	for name, check := range h.checks {
		h.handler.AddLivenessCheck(name, check)
	}
}

func (h *Handler) AddCheck(name string, check healthcheck.Check) {
	h.checks[name] = check
}

func (h *Handler) GetLiveEndpoint() func(http.ResponseWriter, *http.Request) {
	return h.handler.LiveEndpoint
}

func (h *Handler) GetReadyEndpoint() func(http.ResponseWriter, *http.Request) {
	return h.handler.ReadyEndpoint
}

func (h *Handler) SetMaxRoutineCount(cnt int) {
	h.routineCount = cnt
}

func (h *Handler) AddDnsCheck(name string, host string) {
	h.dnsCheck[name] = host
}

func (h *Handler) AddDBCheck(db *gorm.DB) {
	h.db = db
}
