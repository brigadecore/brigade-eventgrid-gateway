package eventgrid

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEvent makes sure we can unmarshal events correctly
func TestEvent(t *testing.T) {
	tests := []string{
		"testdata/spec-json-01.json",
		"testdata/spec-json-02.json",
		"testdata/spec-json-03.json",
	}

	for _, file := range tests {
		raw, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		events := new([]*Event)
		if err = json.Unmarshal(raw, events); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewFromRequest(t *testing.T) {
	is := assert.New(t)

	data, err := ioutil.ReadFile("testdata/spec-json-01.json")
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewReader(data)

	ev, err := NewFromRequestBody(body)
	if err != nil {
		t.Fatal(err)
	}

	is.Equal("Microsoft.Storage.BlobCreated", ev.EventType)
	is.Equal("/blobServices/default/containers/oc2d2817345i200097container/blobs/oc2d2817345i20002296blob", ev.Subject)
}
