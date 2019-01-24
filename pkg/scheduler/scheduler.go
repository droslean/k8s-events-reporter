package scheduler

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/droslean/events-reporter/pkg/config"
	"github.com/droslean/events-reporter/pkg/controller"
)

// Scheduler holds the details for the scheduler.
type Scheduler struct {
	name           string
	lastReportTime metav1.Time
	report         config.Report
	fieldSets      []fields.Set
	client         corev1.CoreV1Interface
	mutex          *sync.Mutex
	receiverChan   chan controller.Email
	events         []v1.Event
	logger         *logrus.Entry
	email          controller.Email
}

// NewScheduler creates a new scheduler.
func NewScheduler(name string, report config.Report, client corev1.CoreV1Interface, receiverChan chan controller.Email) *Scheduler {
	return &Scheduler{
		name:           name,
		lastReportTime: metav1.Now(),
		report:         report,
		fieldSets:      getFieldSets(report.Kind, report.Reasons),
		client:         client,
		mutex:          &sync.Mutex{},
		receiverChan:   receiverChan,
		logger:         logrus.WithFields(logrus.Fields{"sched_name": name, "desc": report.Description}),
	}
}

// Start starts the scheduler and gets the events every report.interval.
func (s *Scheduler) Start(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer utilruntime.HandleCrash()
	defer wg.Done()

	s.logger.Info("Started")
	go wait.Until(s.getEvents, s.report.Interval, stopCh)

	<-stopCh
}

func (s *Scheduler) getEvents() {
	var events []v1.Event
	var body []string

	s.email = controller.Email{
		Recipients: s.report.EmailRecipients,
	}

	for _, field := range s.fieldSets {
		e, err := s.client.Events(metav1.NamespaceAll).List(metav1.ListOptions{
			FieldSelector: field.AsSelector().String()})
		if err != nil {
			s.logger.WithError(err).Error("could not get events")
			return
		}
		events = append(events, e.Items...)
	}

	s.events = s.filterEventTimeRange(events)
	if len(s.events) > 0 {
		for _, event := range s.events {
			body = append(body, fmt.Sprintf("Name:%s\nKind:%s\nType:%s\nReason:%s\nMessage:%s\nLastTimestamp:%s\n",
				event.Name, event.InvolvedObject.Kind, event.Type, event.Reason, event.Message, event.LastTimestamp))
		}
	} else {
		s.reportEvents(controller.Email{})
		s.logger.WithField("since", s.lastReportTime).Info("No events were found for the specified time range")
	}

	if len(body) > 0 {
		s.email.Body = body
		s.email.Subject = fmt.Sprintf("Event report: %s since [%s]", s.report.Description, s.lastReportTime)
		s.reportEvents(s.email)
	}
	s.lastReportTime = metav1.Now()
}

func getFieldSets(involvedObjectKind string, reasons []string) []fields.Set {
	var fieldSets []fields.Set
	for _, reason := range reasons {
		field := fields.Set{}
		if len(involvedObjectKind) > 0 {
			field["involvedObject.kind"] = involvedObjectKind
		}
		if len(reason) > 0 {
			field["reason"] = reason
		}
		fieldSets = append(fieldSets, field)
	}
	return fieldSets
}

func (s *Scheduler) reportEvents(report controller.Email) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.receiverChan <- report
}

func (s *Scheduler) filterEventTimeRange(events []v1.Event) []v1.Event {
	var allEvents []v1.Event
	for _, event := range events {
		if s.lastReportTime.Before(&event.LastTimestamp) || s.lastReportTime == event.LastTimestamp {
			allEvents = append(allEvents, event)
		}
	}
	return allEvents
}
