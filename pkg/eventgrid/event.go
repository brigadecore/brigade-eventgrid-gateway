package eventgrid

import (
	"encoding/json"
	"io"
	"time"
)

const (
	// ValidationEvent is raised when Eventgrid tries to validate an endpoint
	ValidationEvent = "Microsoft.EventGrid.SubscriptionValidationEvent"
)

// Event represents an Azure EventGrid event
//
// Events consist of a set of five required string properties and a required data object.
// The properties are common to all events from any publisher.
// The data object contains properties that are specific to each publisher.
type Event struct {
	// Topic represents resource path to the event source.
	// This field is not writeable. Event Grid provides this value.
	Topic string `json:"topic"`
	// Subject is the publisher-defined path to the event subject.
	Subject string `json:"subject"`
	// EventType is one of the registered event types for this event source.
	EventType string `json:"eventType"`
	// EventTime is the time the event is generated based on the provider's UTC time.
	EventTime time.Time `json:"eventTime"`
	// ID is the unique identifier for the event.
	ID string `json:"id"`
	// Data is the event data specific to the resource provider.
	Data interface{} `json:"data"`
	// DataVersion is the schema version of the data object.
	// The publisher defines the schema version.
	DataVersion string `json:"dataVersion"`
	// Metadata is the schema version of the event metadata.
	// Event Grid defines the schema of the top-level properties.
	// Event Grid provides this value.
	MetadataVersion string `json:"metadataVersion"`
}

// NewFromRequestBody decodes the body of an HTTP request and returns a single event
//
// Event Grid sends the events to subscribers in an array that contains a single event.
// This behavior may change in the future.
func NewFromRequestBody(body io.Reader) (*Event, error) {
	events := new([]*Event)

	decoder := json.NewDecoder(body)
	err := decoder.Decode(&events)
	if err != nil {
		return nil, err
	}

	// EventGrid sends an array containing a single event
	// we return the only item in the array
	return (*events)[0], nil
}
