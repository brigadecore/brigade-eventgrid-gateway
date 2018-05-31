package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/brigade/pkg/brigade"
	"github.com/Azure/brigade/pkg/storage"
	"github.com/Azure/brigade/pkg/storage/mock"

	"github.com/radu-matei/brigade-eventgrid-gateway/pkg/cloudevents"
	"github.com/radu-matei/brigade-eventgrid-gateway/pkg/eventgrid"
)

func TestHeahtlz(t *testing.T) {
	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	router := setupRouter(nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("wrong status code: got %v, expected %v", status, http.StatusOK)
	}

	expected := `{"message":"ok"}`
	if body := rr.Body.String(); body != expected {
		t.Errorf("wrong body: got %v, expected %v", body, expected)
	}
}

func TestValidation(t *testing.T) {
	raw, err := ioutil.ReadFile("testdata/validation.json")
	if err != nil {
		t.Fatal(err)
	}

	ev, err := eventgrid.NewFromRequestBody(bytes.NewBuffer(raw))
	if err != nil {
		t.Fatal(err)
	}

	data := ev.Data.(map[string]interface{})

	response := make(map[string]interface{})
	response["validationResponse"] = data["validationCode"]

	expected, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}

	paths := []string{
		eventGridPath,
		cloudEventsPath,
	}

	for _, p := range paths {
		req, err := http.NewRequest("POST", p, bytes.NewBuffer(raw))
		if err != nil {
			t.Fatal(err)
		}

		s := setupStore()

		router := setupRouter(s)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("wrong status code: got %v, expected %v", status, http.StatusOK)
		}

		if body := rr.Body.String(); body != string(expected) {
			t.Errorf("wrong body: got %v, expected %v", body, string(expected))
		}
	}
}

func TestEventGrid(t *testing.T) {
	files := []string{
		"testdata/eventgrid-blob-created.json",
	}

	for _, file := range files {
		raw, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		ev, err := eventgrid.NewFromRequestBody(bytes.NewBuffer(raw))
		if err != nil {
			t.Fatal(err)
		}

		expected, err := json.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", eventGridPath, bytes.NewBuffer(raw))
		if err != nil {
			t.Fatal(err)
		}

		testRequest(t, req, string(expected))
	}
}

func TestCloudEvents(t *testing.T) {
	files := []string{
		"testdata/cloudevents-blob-created.json",
	}

	for _, file := range files {
		raw, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", cloudEventsPath, bytes.NewBuffer(raw))
		if err != nil {
			t.Fatal(err)
		}
		// the request header must contain the CloudEvents content type
		req.Header.Set("content-type", cloudevents.CloudEventsContentType)

		ev, err := cloudevents.NewFromRequest(req)
		if err != nil {
			t.Fatal(err)
		}

		// put request body back so we can pass it to testRequest
		req.Body = ioutil.NopCloser(bytes.NewBuffer(raw))

		expected, err := json.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}

		testRequest(t, req, string(expected))
	}
}

// testRequest creates a new HTTP request using the JSON payload in testdata/
// and checks for status code, return body and for creation of the build in the store
//
// The test assumes the build payload is the actual event received by the gateway
func testRequest(t *testing.T, req *http.Request, expected string) {
	// setup mock Brigade store
	s := setupStore()

	router := setupRouter(s)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// check HTTP status
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("wrong status code: got %v, expected %v", status, http.StatusOK)
	}

	// check return body
	if body := rr.Body.String(); body != expected {
		t.Errorf("wrong body: got %v, expected %v", body, expected)
	}

	p, err := s.GetProject(projectID)
	if err != nil {
		t.Errorf("cannot get projects: %v", err)
	}

	b, err := s.GetProjectBuilds(p)
	if err != nil {
		t.Errorf("cannot get project builds: %v", err)
	}

	// this is the only build in the mock store
	// check if the payload is the actual event received
	actualPayload := string(b[0].Payload)
	if actualPayload != expected {
		t.Errorf("wrong build payload: expected %v, got %v", expected, actualPayload)
	}
}

func setupStore() storage.Store {
	s := mock.New()
	s.Project = &brigade.Project{
		ID: projectID,
		Secrets: map[string]string{
			"eventGridToken": token,
		},
	}

	return s
}

const (
	projectID = "project-id"
	token     = "super-secret-token"
)

var (
	eventGridPath   = fmt.Sprintf("/eventgrid/%s/%s", projectID, token)
	cloudEventsPath = fmt.Sprintf("/cloudevents/v0.1/%s/%s", projectID, token)
)
