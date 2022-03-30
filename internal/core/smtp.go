package core

import (
	"context"
	"log"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type SMTPService struct {
	cfg  *viper.Viper
	smtp *smtp.Server
}

const (
	maxMessageBytes = 1 << 20
	maxRecipients   = 50
)

func NewSMTPService(cfg *viper.Viper, be smtp.Backend) *SMTPService {
	s := smtp.NewServer(be)

	s.Addr = cfg.GetString("smtp.addr")
	s.Domain = cfg.GetString("smtp.domain")
	s.MaxMessageBytes = maxMessageBytes
	s.MaxRecipients = maxRecipients
	s.AllowInsecureAuth = true

	return &SMTPService{
		cfg:  cfg,
		smtp: s,
	}
}

func (s *SMTPService) Run(ctx context.Context) error {
	addr := s.cfg.GetString("smtp.addr")
	errC := make(chan error, 1)
	go func() {
		err := s.smtp.ListenAndServe()
		if strings.Contains(err.Error(), "use of closed network connection") {
			err = nil
		}
		errC <- errors.WithStack(err)
	}()

	select {
	case <-ctx.Done():
		log.Printf("stop smtp server, address: %s, error: %s", addr, ctx.Err())

		err := s.smtp.Close()
		if err != nil {
			log.Printf("failed to shutdown smtp server at address: %s, error: %s", addr, err)
		}

		return nil
	case err := <-errC:
		return err
	}
}
