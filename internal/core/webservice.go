package core

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type WebService struct {
	cfg     *viper.Viper
	handler http.Handler
}

func NewWebService(cfg *viper.Viper, handler http.Handler) *WebService {
	return &WebService{
		cfg:     cfg,
		handler: handler,
	}
}

func (s *WebService) Run(ctx context.Context) error {
	addr := s.cfg.GetString("http.addr")
	server := &http.Server{
		Addr:         addr,
		Handler:      s.handler,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		ErrorLog:     log.New(os.Stdout, "HTTP Main Mux: ", log.Llongfile),
	}

	errC := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		errC <- errors.WithStack(err)
	}()

	select {
	case <-ctx.Done():
		log.Printf("stop http server, address: %s, error: %s", addr, ctx.Err())

		timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := server.Shutdown(timeout)
		if err != nil {
			log.Printf("failed to shutdown http server at address: %s, error: %s", addr, err)
		}

		return nil
	case err := <-errC:
		return err
	}
}
