package main

import (
	"context"
	"log"
	"net/http"

	"smtp2postmanq/internal/amqp"
	"smtp2postmanq/internal/config"
	"smtp2postmanq/internal/core"
	"smtp2postmanq/internal/healthcheck"
	"smtp2postmanq/internal/smtp"
)

func main() {
	ctx := context.Background()

	hc := healthcheck.Provide()
	hc.SetMaxRoutineCount(200)

	cfg := config.Load()

	amqpProvider := amqp.Provide(cfg, hc)

	hc.Init()

	app := core.NewApplication()

	mainMux := http.NewServeMux()
	mainMux.Handle("/_health", hc.GetEndpoint())

	app.Register(core.NewWebService(cfg, mainMux))
	app.Register(core.NewSMTPService(cfg, smtp.NewBackend(cfg, amqpProvider)))
	app.Register(amqpProvider)

	err := app.Run(ctx)
	if err != nil {
		log.Fatalf("application error: %s", err)
		return
	}

	log.Print("application stopped")
}
