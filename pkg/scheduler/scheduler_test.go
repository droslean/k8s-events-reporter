package scheduler

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/droslean/events-reporter/pkg/config"
	"github.com/droslean/events-reporter/pkg/controller"
)

type schedTestingClient struct {
	kubecs *fake.Clientset
	t      *testing.T
}

func (c *schedTestingClient) Core() corev1.CoreV1Interface {
	fc := c.kubecs.Core().(*fakecorev1.FakeCoreV1)
	return &schedTestingCore{*fc, c.t}
}

type schedTestingCore struct {
	fakecorev1.FakeCoreV1
	t *testing.T
}

func (c *schedTestingCore) Events(ns string) corev1.EventInterface {
	events := c.FakeCoreV1.Events(ns).(*fakecorev1.FakeEvents)
	return &schedTestingEvents{*events, c.t}
}

type schedTestingEvents struct {
	fakecorev1.FakeEvents
	t *testing.T
}

func TestGetEvents(t *testing.T) {
	wg := &sync.WaitGroup{}

	testCase := struct {
		event    *v1.Event
		report   config.Report
		expected controller.Email
	}{
		event: &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			InvolvedObject: v1.ObjectReference{
				Kind:       "Pod",
				Name:       "foo",
				UID:        "bar",
				APIVersion: "version",
				FieldPath:  "spec.containers[2]",
			},
			Reason:        "Started",
			Message:       "some verbose message: 1",
			Source:        v1.EventSource{Component: "eventTest"},
			Count:         1,
			Type:          v1.EventTypeNormal,
			LastTimestamp: metav1.NewTime(time.Date(2019, time.January, 15, 0, 0, 0, 0, time.UTC)),
		},
		report: config.Report{
			Description:     "test scheduler",
			Kind:            "Pod",
			Reasons:         []string{"Started"},
			EmailRecipients: []string{"test@test.com"},
		},
		expected: controller.Email{
			Recipients: []string{"test@test.com"},
			Body:       []string{"Name:foo\nKind:Pod\nType:Normal\nReason:Started\nMessage:some verbose message: 1\nLastTimestamp:2019-01-15 00:00:00 +0000 UTC\n"},
			Subject:    "Event report: test scheduler since [2019-01-14 00:00:00 +0000 UTC]",
		},
	}

	client := schedTestingClient{
		kubecs: fake.NewSimpleClientset(),
		t:      t,
	}

	receiverChan := make(chan controller.Email)

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case report := <-receiverChan:
			if !reflect.DeepEqual(report, testCase.expected) {
				t.Fatalf("Found:\n%v\nExpected:%v\n", report, testCase.expected)
			}

		}
	}()

	scheduler := NewScheduler("test_sched", testCase.report, client.kubecs.CoreV1(), receiverChan)
	scheduler.lastReportTime = metav1.NewTime(time.Date(2019, time.January, 14, 0, 0, 0, 0, time.UTC))

	eventClient := client.Core()
	_, err := eventClient.Events(metav1.NamespaceAll).Create(testCase.event)
	if err != nil {
		t.Fatalf("could not create event: %v", err)
	}

	scheduler.getEvents()
	wg.Wait()
}

func TestGetFieldSets(t *testing.T) {
	testCases := []struct {
		valid              bool
		involvedObjectKind string
		reasons            []string
		expected           []fields.Set
	}{
		{
			valid:              true,
			involvedObjectKind: "Pod",
			reasons:            []string{"Started", "Created"},
			expected: []fields.Set{
				{
					"involvedObject.kind": "Pod",
					"reason":              "Started",
				},
				{
					"involvedObject.kind": "Pod",
					"reason":              "Created",
				},
			},
		},
		{
			valid:              false,
			involvedObjectKind: "Pod",
			reasons:            []string{"Started", "Unknown"},
			expected: []fields.Set{
				{
					"involvedObject.kind": "Pod",
					"reason":              "Started",
				},
				{
					"involvedObject.kind": "Pod",
					"reason":              "Created",
				},
			},
		},
		{
			valid:              true,
			involvedObjectKind: "Deployment",
			reasons:            []string{"ScalingReplicaSet", "Created", "Test1", "Test2"},
			expected: []fields.Set{
				{
					"involvedObject.kind": "Deployment",
					"reason":              "ScalingReplicaSet",
				},
				{
					"involvedObject.kind": "Deployment",
					"reason":              "Created",
				},
				{
					"involvedObject.kind": "Deployment",
					"reason":              "Test1",
				},
				{
					"involvedObject.kind": "Deployment",
					"reason":              "Test2",
				},
			},
		},
	}

	for _, testCase := range testCases {
		fieldSets := getFieldSets(testCase.involvedObjectKind, testCase.reasons)
		if !reflect.DeepEqual(testCase.expected, fieldSets) == testCase.valid {
			t.Fatalf("Expected %v\nFound %v", testCase.expected, fieldSets)
		}
	}
}

func TestFilterEventTimeRange(t *testing.T) {
	scheduler := &Scheduler{
		lastReportTime: metav1.NewTime(time.Date(2019, time.January, 15, 0, 0, 0, 0, time.UTC)),
	}

	events := []v1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			LastTimestamp: metav1.NewTime(time.Date(2019, time.January, 14, 0, 0, 0, 0, time.UTC)),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo2",
			},
			LastTimestamp: metav1.NewTime(time.Date(2019, time.January, 14, 0, 0, 0, 0, time.UTC)),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo3",
			},
			LastTimestamp: metav1.NewTime(time.Date(2019, time.January, 14, 0, 0, 0, 0, time.UTC)),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo4",
			},
			LastTimestamp: metav1.NewTime(time.Date(2019, time.January, 15, 0, 0, 0, 0, time.UTC)),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo5",
			},
			LastTimestamp: metav1.NewTime(time.Date(2019, time.January, 15, 0, 0, 0, 0, time.UTC)),
		},
	}

	expected := []v1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo4",
			},
			LastTimestamp: metav1.NewTime(time.Date(2019, time.January, 15, 0, 0, 0, 0, time.UTC)),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo5",
			},
			LastTimestamp: metav1.NewTime(time.Date(2019, time.January, 15, 0, 0, 0, 0, time.UTC)),
		},
	}

	if e := scheduler.filterEventTimeRange(events); !reflect.DeepEqual(expected, e) {
		t.Fatalf("Expected: %v\nFound: %v", expected, e)
	}
}
