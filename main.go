package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"smtp2postmanq/amqp"
	"smtp2postmanq/config"
	"smtp2postmanq/healthcheck"
	"smtp2postmanq/http"
	"smtp2postmanq/shutdown"

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
func (be *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.User, error) {
	// test base 64 auth AHNtdHBfbG9naW4Ac210cF9wYXNzd29yZA==
	if username != be.cfg.GetString("smtp.login") || password != be.cfg.GetString("smtp.password") {
		return nil, errors.New("invalid username or password")
	}
	return &User{cfg: be.cfg, amqp: be.amqp}, nil
}

// AnonymousLogin requires clients to authenticate using SMTP AUTH before sending emails
func (be *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.User, error) {
	fmt.Println(state)
	return nil, smtp.ErrAuthRequired
}

// A User is returned after successful login.
type User struct {
	cfg *viper.Viper
	amqp amqp.IAMQP
}

// Send handles an email.
func (u *User) Send(from string, to []string, r io.Reader) (err error) {
	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(r); err != nil {
		return
	} else {
		var recepient string
		if len(to) > 0 {
			recepient = to[0]
		}
		err = u.amqp.SendEmailToQueue(amqp.SendMail{
			Envelop:from,
			Recipient:recepient,
			Body:buf.String(),
		})
		if err != nil {
			return
		}
	}
	return
}

// Logout handles logout.
func (u *User) Logout() error {
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
	s.MaxIdleSeconds = 300
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	go func(){
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
