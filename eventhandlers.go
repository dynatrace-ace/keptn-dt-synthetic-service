package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
)

/**
* Here are all the handler functions for the individual event
* See https://github.com/keptn/spec/blob/0.8.0-alpha/cloudevents.md for details on the payload
**/

type SyntheticEventData struct {
	keptnv2.EventData
	MonitorId string `json:"monitorId"`
}

type SyntheticSuccessfulEventData struct {
	keptnv2.EventData
	SyntheticExecution SyntheticExecution
}

type SyntheticExecution struct {
	BatchId      string   `json:"batchId"`
	ExecutionIds []string `json:"executionIds"`
}

type ExecutionResponseBody struct {
	BatchId           string                  `json:"batchId"`
	NotTriggeredCount int16                   `json:"notTriggeredCount"`
	NotTriggered      []ExecutionNotTriggered `json:"notTriggered"`
	TriggeredCount    int16                   `json:"triggeredCount"`
	Triggered         []ExecutionTriggered    `json:"triggered"`
}

type ExecutionNotTriggered struct {
	MonitorId string `json:"monitorId"`
	Cause     string `json:"cause"`
}

type ExecutionTriggered struct {
	MonitorId  string `json:"monitorId"`
	Executions []struct {
		ExecutionId string `json:"executionId"`
		LocationId  string `json:"locationId"`
	} `json:"executions"`
}

func SyntheticCloudEventHandler(myKeptn *keptnv2.Keptn, incomingEvent cloudevents.Event, data *SyntheticEventData, httpClient *http.Client) error {
	log.Printf("Handling %s Event: %s", incomingEvent.Type(), incomingEvent.Context.GetID())

	// marshaledData, _ := json.MarshalIndent(data, "", "  ")
	// log.Println(string(marshaledData))

	// marshaledEvent, _ := json.MarshalIndent(incomingEvent, "", "  ")
	// log.Println(string(marshaledEvent))

	_, err := myKeptn.SendTaskStartedEvent(data, ServiceName)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send task started CloudEvent (%s), aborting...", err.Error())
		log.Println(errMsg)
		return err
	}

	var jsonData = []byte(fmt.Sprintf(`{
		"monitorsToTrigger": [{
			"monitorId": "%s",
      "locations": []
		}]
	}`, data.MonitorId))

	if err != nil {
		errMsg := fmt.Sprint("Failed to marshal POST body, aborting...", err.Error())
		log.Println(errMsg)

		_, err = myKeptn.SendTaskFinishedEvent(&keptnv2.EventData{
			Status:  keptnv2.StatusErrored, // alternative: keptnv2.StatusErrored
			Result:  keptnv2.ResultFailed,  // alternative: keptnv2.ResultFailed
			Message: "Failed to marshal POST body!",
		}, ServiceName)

		return err
	}

	u, err := url.Parse(os.Getenv("DT_TENANT"))
	if err != nil {
		errMsg := fmt.Sprint("Failed to parse DT_TENANT from environment, aborting...", err.Error())
		log.Println(errMsg)

		_, err = myKeptn.SendTaskFinishedEvent(&keptnv2.EventData{
			Status:  keptnv2.StatusErrored, // alternative: keptnv2.StatusErrored
			Result:  keptnv2.ResultFailed,  // alternative: keptnv2.ResultFailed
			Message: "Failed to parse DT_TENANT!",
		}, ServiceName)

		return err
	}

	u.Path = path.Join(u.Path, "/api/v2/synthetic/monitors/execute")

	executeUrl := u.String()
	dtApiToken := os.Getenv("DT_API_TOKEN")

	req, err := http.NewRequest("POST", executeUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		errMsg := fmt.Sprint("Failed to generate Synthetic monitor request, aborting...", err.Error())
		log.Println(errMsg)

		_, err = myKeptn.SendTaskFinishedEvent(&keptnv2.EventData{
			Status:  keptnv2.StatusErrored,
			Result:  keptnv2.ResultFailed,
			Message: "Failed to trigger Synthetic monitor!",
		}, ServiceName)

		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Api-Token %s", dtApiToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		errMsg := fmt.Sprint("Failed to trigger Synthetic monitor, aborting...", err.Error())
		log.Println(errMsg)

		_, err = myKeptn.SendTaskFinishedEvent(&keptnv2.EventData{
			Status:  keptnv2.StatusErrored,
			Result:  keptnv2.ResultFailed,
			Message: "Failed to trigger Synthetic monitor!",
		}, ServiceName)

		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	executionResponseBody := ExecutionResponseBody{}
	err = json.Unmarshal(body, &executionResponseBody)

	if err != nil {
		errMsg := fmt.Sprint("Failed to read Synthetic monitor response, aborting...", err.Error())
		log.Println(errMsg)

		_, err = myKeptn.SendTaskFinishedEvent(&keptnv2.EventData{
			Status:  keptnv2.StatusErrored,
			Result:  keptnv2.ResultFailed,
			Message: "Failed to trigger Synthetic monitor!",
		}, ServiceName)

		return err
	}

	if executionResponseBody.NotTriggeredCount > 0 {
		errMsg := fmt.Sprint("Failed to trigger Synthetic monitor, aborting...", executionResponseBody.NotTriggered)
		log.Println(errMsg)

		_, err = myKeptn.SendTaskFinishedEvent(&keptnv2.EventData{
			Status:  keptnv2.StatusErrored,
			Result:  keptnv2.ResultFailed,
			Message: "Failed to trigger Synthetic monitor!",
		}, ServiceName)

		return err
	}

	syntheticExecution := SyntheticExecution{
		BatchId:      executionResponseBody.BatchId,
		ExecutionIds: []string{},
	}

	for _, triggered := range executionResponseBody.Triggered {
		for _, execution := range triggered.Executions {
			syntheticExecution.ExecutionIds = append(syntheticExecution.ExecutionIds, execution.ExecutionId)
		}
	}

	// TBD: Wait for Synthetic trigger results

	syntheticSuccessfulEventData := &SyntheticSuccessfulEventData{
		EventData: keptnv2.EventData{
			Status: keptnv2.StatusSucceeded,
			Result: keptnv2.ResultPass,
		},
		SyntheticExecution: syntheticExecution,
	}

	_, err = myKeptn.SendTaskFinishedEvent(syntheticSuccessfulEventData, ServiceName)

	return err
}
