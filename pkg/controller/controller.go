package controller

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	gomail "gopkg.in/gomail.v2"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/droslean/events-reporter/pkg/config"
)

// Controller holds information for the controller.
type Controller struct {
	Receiver chan Email

	emailSettings config.EmailSettings
	logger        *logrus.Entry
	message       *gomail.Message
}

// Email holds details for the email that will be send to the report recipients.
type Email struct {
	Recipients []string
	Body       []string
	Subject    string
}

// NewController creates a new controller.
func NewController(emailSettings config.EmailSettings) *Controller {
	return &Controller{
		emailSettings: emailSettings,
		Receiver:      make(chan Email),
		logger:        logrus.WithField("component", "controller"),
	}
}

// Start ...
func (c *Controller) Start(stop chan struct{}, wg *sync.WaitGroup) {
	defer utilruntime.HandleCrash()
	defer wg.Done()
	for {
		select {
		case report := <-c.Receiver:
			empty := reflect.New(reflect.TypeOf(report)).Elem().Interface()
			if !reflect.DeepEqual(report, empty) {
				c.logger.WithField("subject", report.Subject).Info("report received")
				if err := c.sendEmail(report); err != nil {
					c.logger.WithError(err).Error("couldn not send email")
				}
			}
		case <-stop:
			return
		}
	}
}

func (c *Controller) sendEmail(email Email) error {
	c.message = gomail.NewMessage()
	c.message.SetHeader("From", "event-reporter@test.com")
	c.message.SetHeader("To", email.Recipients...)
	c.message.SetHeader("Subject", email.Subject)
	c.message.SetBody("text/html", strings.Join(email.Body, "\n"))

	if len(c.emailSettings.SMTPServer) == 0 {
		return errors.New("no smtp server is configured")
	}

	if c.emailSettings.Port == 0 {
		return errors.New("no port specified in email settings")
	}

	dialer := gomail.NewDialer(c.emailSettings.SMTPServer, c.emailSettings.Port, c.emailSettings.Username, c.emailSettings.Password)
	if err := dialer.DialAndSend(c.message); err != nil {
		return err
	}
	return nil
}
