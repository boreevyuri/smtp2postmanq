package smtp

import (
	"errors"
	"io"
	"io/ioutil"

	"github.com/emersion/go-smtp"
	"github.com/spf13/viper"

	"smtp2postmanq/internal/amqp"
)

type IAMQP interface {
	SendEmailToQueue(send *amqp.SendMail) error
}

// The Backend implements SMTP server methods.
type Backend struct {
	cfg  *viper.Viper
	amqp IAMQP
}

func NewBackend(cfg *viper.Viper, amqp IAMQP) *Backend {
	return &Backend{
		cfg:  cfg,
		amqp: amqp,
	}
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
	amqp IAMQP
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
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	for _, r := range s.msg.To {
		err = s.amqp.SendEmailToQueue(&amqp.SendMail{
			Envelop:   s.msg.From,
			Recipient: r,
			Body:      b,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
