package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"smtp2postmanq/internal/amqp"
	"smtp2postmanq/internal/config"
	"smtp2postmanq/internal/healthcheck"
	"smtp2postmanq/internal/http"
	"smtp2postmanq/internal/shutdown"

	"github.com/emersion/go-smtp"
	"github.com/spf13/viper"
)

// The Backend implements SMTP server methods.
type Backend struct {
	cfg  *viper.Viper
	hc   healthcheck.IHealthHandler
	amqp amqp.IAMQP
}

// Login handles a login command with username and password.
func (be *Backend) Login(_ *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	if username != be.cfg.GetString("smtp.login") || password != be.cfg.GetString("smtp.password") {
		return nil, errors.New("invalid username or password")
	}
	return &Session{cfg: be.cfg, amqp: be.amqp}, nil
}

// AnonymousLogin requires clients to authenticate using SMTP AUTH before sending emails
func (be *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return nil, smtp.ErrAuthRequired
}

// A Session is returned after successful login.
type Session struct {
	cfg  *viper.Viper
	amqp amqp.IAMQP
	msg  *message
}

type message struct {
	From string
	To   []string
	Data []byte
	Opts smtp.MailOptions
}

// Logout handles logout.
func (s *Session) Logout() error {
	return nil
}

// Reset message.
func (s *Session) Reset() {
	s.msg = &message{}
}

// Mail init mailing
func (s *Session) Mail(from string, opts smtp.MailOptions) error {
	s.Reset()
	s.msg.From = from
	s.msg.Opts = opts
	return nil
}

// Rcpt add recepient
func (s *Session) Rcpt(to string) error {
	s.msg.To = append(s.msg.To, to)
	return nil
}

// Data add message body and send message
func (s *Session) Data(r io.Reader) error {
	if b, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		for _, r := range s.msg.To {
			err = s.amqp.SendEmailToQueue(amqp.SendMail{
				Envelop:   s.msg.From,
				Recipient: r,
				Body:      b,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	hc := healthcheck.ProvideHandler()
	hc.Init()
	hc.SetMaxRoutineCount(200)

	cfg := config.Load()

	shutDown := shutdown.Provide()

	be := &Backend{cfg: cfg, hc: hc, amqp: amqp.Provide(cfg, hc, shutDown)}

	s := smtp.NewServer(be)

	s.Addr = be.cfg.GetString("smtp.addr")
	s.Domain = be.cfg.GetString("smtp.domain")
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	go func() {
		log.Println("Starting smtp server at", s.Addr)

		if err := s.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			log.Print(err)
		}
		log.Printf("SMTP server stopped on %q", s.Addr)
		shutDown.ShutdownSuccess()
	}()
	go func() {
		if shutDown.IsShutdown() {
			s.Close()
		}
	}()

	done := make(chan error, 1)
	server := http.GetServer(cfg, hc, shutDown)
	server.ListenAndServe(done)
	if err := <-done; err != nil {
		panic(err)
	}
}
