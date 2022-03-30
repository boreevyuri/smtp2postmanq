package amqp

//go:generate easyjson

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/alexliesenfeld/health"

	"github.com/spf13/viper"
	"github.com/streadway/amqp"

	"smtp2postmanq/internal/healthcheck"
)

var (
	ConnectionClosed = errors.New("connection closed")
)

//easyjson:json
type SendMail struct {
	Envelop   string `json:"envelope"`
	Recipient string `json:"recipient"`
	Body      []byte `json:"body"`
}

type Provider struct {
	conn               *amqp.Connection
	channel            *amqp.Channel
	cfg                *viper.Viper
	healthCheckHandler healthcheck.IHealthHandler
}

func Provide(cfg *viper.Viper, healthCheckHandler healthcheck.IHealthHandler) *Provider {
	que := &Provider{
		cfg:                cfg,
		conn:               new(amqp.Connection),
		healthCheckHandler: healthCheckHandler,
	}

	err := que.connect()
	if err != nil {
		log.Panic(err)
	}

	que.healthCheckInit()

	return que
}

func (que *Provider) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			err := que.channel.Close()
			que.channel = nil
			if err != nil {
				return err
			}

			err = que.conn.Close()
			que.conn = nil
			if err != nil {
				return err
			}

			return nil
		case err := <-que.conn.NotifyClose(make(chan *amqp.Error)):
			que.conn = nil
			que.channel = nil

			return err
		}
	}
}

func (que *Provider) connect() (err error) {
	authString := ""
	login := que.cfg.GetString("amqp.login")
	if login != "" {
		authString += login
	}

	password := que.cfg.GetString("amqp.password")
	if authString != "" && password != "" {
		authString += ":" + password
	}
	if authString != "" {
		authString += "@"
	}

	que.conn, err = amqp.Dial(fmt.Sprintf(
		"amqp://%s%s:%d/%s",
		authString,
		que.cfg.GetString("amqp.host"),
		que.cfg.GetInt("amqp.port"),
		que.cfg.GetString("amqp.virtual_host"),
	))
	if err != nil {
		return
	}

	err = que.initQueue()
	if err != nil {
		return
	}

	return
}

func (que *Provider) initQueue() (err error) {
	que.channel, err = que.conn.Channel()
	if err != nil {
		return
	}

	err = que.channel.ExchangeDeclare(
		que.cfg.GetString("amqp.queue"),
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return
	}

	_, err = que.channel.QueueDeclare(
		que.cfg.GetString("amqp.queue"),
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return
	}

	err = que.channel.QueueBind(
		que.cfg.GetString("amqp.queue"),
		que.cfg.GetString("amqp.routing_key"),
		que.cfg.GetString("amqp.queue"),
		false,
		nil,
	)
	if err != nil {
		return
	}

	return
}

func (que *Provider) checkAMQPConnection() error {
	if que.conn == nil {
		return ConnectionClosed
	}

	return nil
}

func (que *Provider) healthCheckInit() {
	if que.healthCheckHandler != nil {

		que.healthCheckHandler.AddCheck("amqp_check", health.WithCheck(health.Check{
			Name: "goroutine-threshold",
			Check: func(ctx context.Context) error {
				if st := que.conn.ConnectionState(); !st.HandshakeComplete || que.conn == nil {
					return fmt.Errorf("expected to complete a TLS handshake, TLS connection state: %+v", st)
				}

				return nil
			},
		}))
	}
}

func (que *Provider) SendEmailToQueue(send *SendMail) error {
	if que.channel == nil {
		return ConnectionClosed
	}

	body, err := json.Marshal(send)
	if err != nil {
		return err
	}

	return que.channel.Publish(
		que.cfg.GetString("amqp.queue"),
		que.cfg.GetString("amqp.routing_key"),
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
}
