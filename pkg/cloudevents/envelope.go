package cloudevents

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

// CE- header constants, as definied by https://github.com/cloudevents/spec/blob/v0.1/http-transport-binding.md
// Not that these capitalization scheme is incompatible with Go's built-in scheme.
const (
	CECloudEventsVersion = "CE-CloudEventsVersion"
	CEEventType          = "CE-EventType"
	CEEventTypeVersion   = "CE-EventTypeVersion"
	CEEventID            = "CE-EventID"
	CESource             = "CE-Source"
	CEEventTime          = "CE-EventTime"
	CEExtensions         = "CE-Extensions"
)

const (
	//CloudEventsContentType is the content type for a cloud events JSON payload.
	CloudEventsContentType = "application/cloudevents+json"
)

// Envelope is the top-level event object.
//
// Since the gateway's job is to forward this data on to the next hop, we
// are less concerned with parsing everything into the exact fields necessary,
// and more concerned with preserving data for the worker.
//
// See section 2.4 of the JSON spec
// https://github.com/cloudevents/spec/blob/v0.1/json-format.md#24-examples
type Envelope struct {
	// EventType is the event type name.
	// The spec gives the example "com.example.someevent"
	EventType string `json:"eventType"`
	// EventTypeVersion is a version field. We suggest a SemVer version
	EventTypeVersion string `json:"eventTypeVersion"`
	// CloudEventsVersion is the version of CloudEvents that this envelope uses.
	// The spec seems to indicate that 0.1 is currently the appropriate version.
	CloudEventsVersion string `json:"cloudEventsVersion"`
	//Source is a URI
	Source string `json:"source"`
	// EventID is the event ID
	EventID string `json:"eventID"`
	// EventTime is the event timestamp in the form 2018-04-05T17:31:00Z (RFC8601?)
	// For Go, we treat this as a string.
	EventTime string `json:"eventTime"`
	// ContentType is the MIME content type of the Data field's payload
	ContentType string `json:"contentType"`
	// Extensions is an arbitrary set of key/value pairs.
	Extensions map[string]interface{} `json:"extensions"`
	// Data is the payload attached to the event.
	// Oddly, the spec provides for both structured and unstructured varieties
	// of Data. The structured variety is JSON data, but schemaless.
	Data interface{} `json:"data"`
}

// NewFromRequest will examine a request and parse appropriately.
func NewFromRequest(req *http.Request) (*Envelope, error) {
	env := new(Envelope)
	// TODO: The spec suggests that +json is not required, but there it also
	// suggests that another format (like Avro) might be used. So we're going
	// with the most conservative reading.
	// https://github.com/cloudevents/spec/blob/v0.1/http-transport-binding.md#3-http-message-mapping
	if ct := req.Header.Get("content-type"); ct != CloudEventsContentType {
		return NewFromHeaders(req)
	}

	body, err := readBody(req)
	if err != nil {
		return env, err
	}

	err = json.Unmarshal(body, env)
	return env, err
}

// NewFromHeaders will construct an Envelope from HTTP headers.
//
// If it is not known whether the headers or the body contain the event,
// use NewFromRequest instead.
func NewFromHeaders(req *http.Request) (*Envelope, error) {
	env := &Envelope{}
	h := req.Header

	// The headers for CE are defined sptrictly to be CE-CamelCase. However, the
	// Go implementation of headers uses an initial caps algo, so CE-CamelCase
	// becomes Ce-Camelcase. But Go's http.Header.Get() does a case insensitive
	// compare. So we use that here.
	if val := h.Get(CECloudEventsVersion); val != "" {
		env.CloudEventsVersion = val
	}
	if val := h.Get(CEEventType); val != "" {
		env.EventType = val
	}
	if val := h.Get(CEEventTypeVersion); val != "" {
		env.EventTypeVersion = val
	}
	if val := h.Get(CEEventID); val != "" {
		env.EventID = val
	}
	if val := h.Get(CESource); val != "" {
		env.Source = val
	}
	if val := h.Get(CEEventTime); val != "" {
		env.EventTime = val
	}
	if val := h.Get("content-type"); val != "" {
		env.ContentType = val
	}

	// According to section 3.1.3 of the spec, any header that beings
	// 'CE-X-' is considered a member of the extensions. In Go's HTTP
	// library, this maps to "Ce-X-".
	env.Extensions = map[string]interface{}{}
	for key, vals := range h {
		prefix := "Ce-X-"
		if strings.HasPrefix(key, prefix) {
			if len(vals) == 0 {
				// In the highly unlikely event that somehow a valueless header
				// got through...
				continue
			}
			// This breaks the spec because the spec conflicts with the HTTP 1.1
			// specification.
			// https://github.com/cloudevents/spec/issues/177
			newkey := strings.ToLower(strings.TrimPrefix(key, prefix))
			env.Extensions[newkey] = vals[0]
		}
	}

	body, err := readBody(req)
	if err != nil {
		return env, err
	}

	// If the content type is JSON, parse the body. Otherwise, copy it as byte
	// data. It's unclear about what MIME types qualify, so we go with the basics.
	if ct := req.Header.Get("content-type"); isJSON(ct) {
		dest := &map[string]interface{}{}
		err := json.Unmarshal(body, dest)
		env.Data = dest
		return env, err
	}

	env.Data = string(body)
	return env, nil
}

func readBody(req *http.Request) ([]byte, error) {
	defer req.Body.Close()
	return ioutil.ReadAll(req.Body)
}

func isJSON(contentType string) bool {
	parts := strings.SplitN(contentType, ";", 2)
	ct := strings.ToLower(parts[0])
	switch ct {
	case "application/json", "text/json":
		return true
	default:
		return strings.HasSuffix(ct, "+json")
	}
}
