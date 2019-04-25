package amqp

//go:generate easyjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"smtp2postmanq/healthcheck"
	"smtp2postmanq/shutdown"
	"smtp2postmanq/utils"

	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

const heathcheckInterval = 5 * time.Second

var (
	ConnectionClosed = errors.New("connection closed")
)

//easyjson:json
type SendMail struct {
	Envelop   string `json:"envelope"`
	Recipient string `json:"recipient"`
	Body      string `json:"body"`
}

type IAMQP interface {
	SendEmailToQueue(send SendMail) error
}

type amqpProvider struct {
	conn               *amqp.Connection
	channel            *amqp.Channel
	cfg                *viper.Viper
	healthcheckHandler healthcheck.IHealthHandler
	shutDown           shutdown.IGracefullShutdown
}

func Provide(cfg *viper.Viper, healthcheckHandler healthcheck.IHealthHandler, shutDown shutdown.IGracefullShutdown) IAMQP {
	que := &amqpProvider{
		cfg:                cfg,
		conn:               new(amqp.Connection),
		healthcheckHandler: healthcheckHandler,
		shutDown:           shutDown,
	}

	err := que.connect()
	if err != nil {
		log.Panic(err)
	}

	que.healthcheckInit()
	que.closer()

	return que
}

func (que *amqpProvider) closer() {
	go func() {
		if que.shutDown.IsShutdown() {
			err := que.channel.Close()
			if err == nil {
				que.channel = nil
			}

			err = que.conn.Close()
			if err == nil {
				que.conn = nil
			}

			que.shutDown.ShutdownSuccess()
		}
	}()
}

func (que *amqpProvider) connect() (err error) {
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

func (que *amqpProvider) initQueue() (err error) {
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

func (que *amqpProvider) initCheckAMQPConnection() {
	go func() {
		for err := range que.conn.NotifyClose(make(chan *amqp.Error)) {
			if err != nil {
				log.Print(err)
			}
			que.conn = nil
			que.channel = nil
		}
	}()
}

func (que *amqpProvider) checkAMQPConnection() error {
	if que.conn == nil {
		return ConnectionClosed
	}

	return nil
}

func (que *amqpProvider) healthcheckInit() {
	if que.healthcheckHandler != nil {
		que.initCheckAMQPConnection()

		healthcheckErr := errors.New("amqp is not ready")

		interval := utils.CreateInterval(func() {
			healthcheckErr = que.checkAMQPConnection()
		}, heathcheckInterval)

		que.healthcheckHandler.AddCheck("amqp_check", func() (err error) {
			interval.Lock()
			defer interval.Unlock()
			return healthcheckErr
		})
	}
}

func (que *amqpProvider) SendEmailToQueue(send SendMail) error {
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
