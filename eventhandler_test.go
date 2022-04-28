package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/keptn/go-utils/pkg/lib/v0_2_0/fake"
	"gotest.tools/assert"

	keptn "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
)

/**
 * loads a cloud event from the passed test json file and initializes a keptn object with it
 */
func initializeTestObjects(eventFileName string) (*keptnv2.Keptn, *cloudevents.Event, error) {
	// load sample event
	eventFile, err := ioutil.ReadFile(eventFileName)
	if err != nil {
		return nil, nil, fmt.Errorf("Cant load %s: %s", eventFileName, err.Error())
	}

	incomingEvent := &cloudevents.Event{}
	err = json.Unmarshal(eventFile, incomingEvent)
	if err != nil {
		return nil, nil, fmt.Errorf("Error parsing: %s", err.Error())
	}

	// Add a Fake EventSender to KeptnOptions
	var keptnOptions = keptn.KeptnOpts{
		EventSender: &fake.EventSender{},
	}
	keptnOptions.UseLocalFileSystem = true
	myKeptn, err := keptnv2.NewKeptn(incomingEvent, keptnOptions)

	return myKeptn, incomingEvent, err
}

func TestSyntheticCloudEventHandler(t *testing.T) {
	myKeptn, incomingEvent, err := initializeTestObjects("test-events/test.triggered.json")
	if err != nil {
		t.Error(err)
		return
	}

	httpClient := &http.Client{}

	specificEvent := &SyntheticEventData{}
	err = incomingEvent.DataAs(specificEvent)
	if err != nil {
		t.Errorf("Error getting keptn event data")
	}

	err = SyntheticCloudEventHandler(myKeptn, *incomingEvent, specificEvent, httpClient)
	if err != nil {
		t.Errorf("Error: " + err.Error())
	}

	assert.Equal(t, len(myKeptn.EventSender.(*fake.EventSender).SentEvents), 2)
	assert.Equal(t, keptnv2.GetStartedEventType("test"), myKeptn.EventSender.(*fake.EventSender).SentEvents[0].Type())
	assert.Equal(t, keptnv2.GetFinishedEventType("test"), myKeptn.EventSender.(*fake.EventSender).SentEvents[1].Type())
}
