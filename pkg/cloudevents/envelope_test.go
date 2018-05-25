package cloudevents

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvelope(t *testing.T) {
	is := assert.New(t)
	// Test to make sure that we can unmarshal according to the spec.
	tests := []struct {
		file              string
		eventID           string
		contentType       string
		stringPayload     string
		structuredPayload map[string]interface{}
	}{
		{
			file:          "testdata/spec-json-01.json",
			eventID:       "A234-1234-1234",
			contentType:   "text/xml",
			stringPayload: "<much wow=\"xml\"/>",
		},
		{
			file:          "testdata/spec-json-02.json",
			eventID:       "B234-1234-1234",
			contentType:   "application/vnd.apache.thrift.binary",
			stringPayload: "... base64 encoded string ...",
		},
		{
			file:          "testdata/spec-json-03.json",
			eventID:       "C234-1234-1234",
			contentType:   "application/json",
			stringPayload: "",
			structuredPayload: map[string]interface{}{
				"appinfoA": "abc",
				"appinfoB": 123,
				"appinfoC": true,
			},
		},
	}
	for _, tt := range tests {
		raw, err := ioutil.ReadFile(tt.file)
		if err != nil {
			t.Fatal(err)
		}
		env := new(Envelope)
		if err := json.Unmarshal(raw, env); err != nil {
			t.Fatal(err)
		}

		// These are hardcoded across all spec examples
		is.Equal(env.CloudEventsVersion, "0.1")
		is.Equal(env.EventType, "com.example.someevent")
		is.Equal(env.EventTypeVersion, "1.0")
		is.Equal(env.Source, "/mycontext")
		is.Equal(env.EventTime, "2018-04-05T17:31:00Z")
		is.Equal(env.Extensions["comExampleExtension"], "value")

		// These change per spec example
		is.Equal(env.ContentType, tt.contentType)
		is.Equal(env.EventID, tt.eventID)

		if len(tt.stringPayload) > 0 {
			is.Equal(env.Data.(string), tt.stringPayload)
		}

		if tt.structuredPayload != nil {
			jd := env.Data.(map[string]interface{})
			is.Equal(jd["appinfoA"], "abc")
			is.Equal(jd["appinfoB"], float64(123))
			is.Equal(jd["appinfoC"], true)
		}
	}
}

func mockHeaderRequest(t *testing.T) *http.Request {
	buf := bytes.NewBufferString("payload")
	req, err := http.NewRequest("POST", "http://localhost/mycontext", buf)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set(CECloudEventsVersion, "0.1")
	req.Header.Set(CEEventType, "com.example.someevent")
	req.Header.Set(CEEventTypeVersion, "1.0")
	req.Header.Set(CEEventID, "aaa-bbb-ccc")
	req.Header.Set(CESource, "/mycontext")
	req.Header.Set(CEEventTime, "2018-04-05T17:31:00Z")
	req.Header.Set("content-type", "text/plain")

	// From the spec.
	// Note that the spec contradicts the HTTP 1.1 spec, and consequently we
	// lose a strong mapping of extension names.
	// https://github.com/cloudevents/spec/issues/177
	req.Header.Set("CE-X-Example", "hello")
	req.Header.Set("CE-X-TestExtension", "goodbye")

	return req
}

func mockJSONRequest(t *testing.T, body []byte) *http.Request {
	buf := bytes.NewBuffer(body)
	req, err := http.NewRequest("POST", "http://localhost/mycontext", buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", CloudEventsContentType)
	return req
}

func TestNewFromHeaders(t *testing.T) {
	req := mockHeaderRequest(t)

	env, err := NewFromHeaders(req)
	if err != nil {
		t.Fatal(err)
	}

	is := assert.New(t)
	is.Equal(env.CloudEventsVersion, "0.1", "Events version")
	is.Equal(env.EventType, "com.example.someevent", "Event type")
	is.Equal(env.EventTypeVersion, "1.0", "Event type version")
	is.Equal(env.EventID, "aaa-bbb-ccc", "ID")
	is.Equal(env.Source, "/mycontext", "source")
	is.Equal(env.EventTime, "2018-04-05T17:31:00Z", "date")
	is.Equal(env.ContentType, "text/plain", "content type")
	is.Equal(env.Data, "payload", "data")
	is.Equal(env.Extensions["example"], "hello", "example extension")
	is.Equal(env.Extensions["testextension"], "goodbye", "test extension")
}

func TestNewFromRequest(t *testing.T) {
	is := assert.New(t)
	// The earlier tests verify the individual parsing routines. This test
	// checks to see that the high-level detection logic works.
	req := mockHeaderRequest(t)
	env, err := NewFromHeaders(req)
	if err != nil {
		t.Fatal(err)
	}
	is.Equal("aaa-bbb-ccc", env.EventID, "event ID should be set from headers")

	// Next, test whether it parses a JSON body correctly.
	data, err := ioutil.ReadFile("testdata/spec-json-01.json")
	if err != nil {
		t.Fatal(err)
	}
	req = mockJSONRequest(t, data)

	env, err = NewFromRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	is.Equal("A234-1234-1234", env.EventID, "Event ID should be set from body")
}

func TestIsJSON(t *testing.T) {
	is := assert.New(t)
	hits := []string{
		"application/json",
		"text/json",
		"application/forever-chewing-bubble-gum+json",
	}
	for _, h := range hits {
		is.True(isJSON(h))
	}
	misses := []string{
		"application/x-streussel-cake",
		"json/application",
		"text/plain+json-hippies",
	}
	for _, m := range misses {
		is.False(isJSON(m))
	}

}
