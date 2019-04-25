package http

import (
	"context"
	"log"
	"net/http"
	"os"

	"smtp2postmanq/healthcheck"
	"smtp2postmanq/shutdown"

	"github.com/spf13/viper"
)

type IServer interface {
	ListenAndServe(done chan<- error)
}

type server struct {
	mainMux  *http.ServeMux
	server   *http.Server
	shutDown shutdown.IGracefullShutdown
}

func GetServer(cfg *viper.Viper, hc healthcheck.IHealthHandler, shutDown shutdown.IGracefullShutdown) IServer {
	mainMux := http.NewServeMux()
	s := &server{
		mainMux: mainMux,
		server: &http.Server{
			Addr:     cfg.GetString("http.addr"),
			Handler:  mainMux,
			ErrorLog: log.New(os.Stdout, "HTTP Main Mux: ", log.Llongfile),
		},
		shutDown: shutDown,
	}

	mainMux.Handle("/live", http.HandlerFunc(hc.GetLiveEndpoint()))
	mainMux.Handle("/ready", http.HandlerFunc(hc.GetReadyEndpoint()))

	return s
}

func (s *server) ListenAndServe(done chan<- error) {
	log.Printf("Main server listening on %q", s.server.Addr)

	go func() {
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Error with http main server. Addr: %q. Error: %+v", s.server.Addr, err)
			done <- err
		}

		log.Printf("Main server stopped on %q", s.server.Addr)
		s.shutDown.ShutdownSuccess()
	}()

	go func() {
		if s.shutDown.IsShutdown() {
			_ = s.server.Shutdown(context.Background())
		}
	}()
}
