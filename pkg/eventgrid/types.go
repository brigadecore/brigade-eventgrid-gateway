package eventgrid

import (
	"time"
)

// Event represents an Azure EventGrid event
type Event struct {
	Topic           string      `json:"topic"`
	Subject         string      `json:"subject"`
	EventType       string      `json:"eventType"`
	EventTime       time.Time   `json:"eventTime"`
	ID              string      `json:"id"`
	Data            interface{} `json:"data"`
	DataVersion     string      `json:"dataVersion"`
	MetadataVersion string      `json:"metadataVersion"`
}

// ValidationEvent is raised when Eventgrid tries to validate an endpoint
var ValidationEvent = "Microsoft.EventGrid.SubscriptionValidationEvent"

// BlobCreated is raised when a blob is created
var BlobCreated = "Microsoft.Storage.BlobCreated"

// BlobDeleted is raised when a blob is deleted
var BlobDeleted = "Microsoft.Storage.BlobDeleted"
