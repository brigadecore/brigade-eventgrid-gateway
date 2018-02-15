package eventgrid

import (
	"time"
)

// Event represents an Azure EventGrid event
type Event struct {
	ID      string `json:"id"`
	Topic   string `json:"topic"`
	Subject string `json:"subject"`
	// based on the event type, data contains specific things
	Data            interface{} `json:"data"`
	EventType       string      `json:"eventType"`
	EventTime       time.Time   `json:"eventTime"`
	MetadataVersion string      `json:"metadataVersion"`
	DataVersion     string      `json:"dataVersion"`
}

// DataValidation contains the validation code
type DataValidation struct {
	ValidationCode string `json:"validationCode"`
}
