package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/Azure/brigade/pkg/brigade"
	"github.com/Azure/brigade/pkg/storage"
	"github.com/Azure/brigade/pkg/storage/kube"

	"github.com/radu-matei/brigade-eventgrid-gateway/pkg/cloudevents"
	"github.com/radu-matei/brigade-eventgrid-gateway/pkg/eventgrid"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

var (
	store storage.Store
	debug bool
)

func init() {
	flag.BoolVar(&debug, "debug", true, "enable verbose output")

	flag.Parse()
	if debug {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {

	client, err := kube.GetClient("", os.Getenv("KUBECONFIG"))
	if err != nil {
		log.Fatalf("cannot get Kubernetes client: %v", err)
	}
	store = kube.New(client, "default")

	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "ok"}) })

	e := router.Group("/eventgrid")
	e.POST("/:project", azFn)
	e.POST("/:project/:token", azFn)

	c := router.Group("/cloudevents/v0.1")
	c.POST("/:project/:token", ceFn)

	router.Run()
}

func azFn(c *gin.Context) {
	defer c.Request.Body.Close()

	ev, err := eventgrid.NewFromRequestBody(c.Request.Body)
	if err != nil {
		log.Debugf("cannot get event from request: %v", err)
	}

	log.Debugf("received event: %v", ev)

	if ev.EventType == eventgrid.ValidationEvent {
		sendValidationResponse(c, ev)
		return
	}

	pid := c.Param("project")
	project, err := store.GetProject(pid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "Resource Not Found"})
		log.Debugf("cannot get project ID: %v", err)
		return
	}
	log.Debugf("found project: %v", project)

	// Note that this will always fail on the old route if a token is set on
	// the project.
	// TODO: Change this when Project.Gateways gets implemented.
	if realToken := project.Secrets["eventGridToken"]; realToken != "" {
		tok := c.Param("token")
		if realToken != tok {
			c.JSON(http.StatusForbidden, gin.H{"status": "Forbidden"})
			log.Debugf("token does not match project's version: %v", err)
			return
		}
	}

	payload, err := json.Marshal(ev)
	if err != nil {
		log.Debugf("failed to marshal event: %v", err)
	}

	build := &brigade.Build{
		ProjectID: pid,
		Type:      ev.EventType,
		Provider:  "eventgrid",
		Payload:   payload,
		Revision: &brigade.Revision{
			Ref:    "master",
			Commit: "HEAD",
		},
	}

	log.Debugf("created build: %v", build)

	err = store.CreateBuild(build)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "Failed to invoke hook"})
		log.Debugf("failed to create build: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OK"})
	return
}

// ceFn is a cloud events handler.
func ceFn(c *gin.Context) {

	// read the request body
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Debugf("cannot read body: %v", err)
	}
	// check for validation event
	if bytes.Contains(body, []byte("Microsoft.EventGrid.SubscriptionValidationEvent")) {
		validate(c, bytes.NewReader(body))
		return
	}

	// put request body back so we can decode the envelope
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	// Decoding here does two things: First, it validates the format, and second
	// it converts all of the accepted formats into a uniform representation.
	envelope, err := cloudevents.NewFromRequest(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Malformed body"})
		log.Debugf("cannot decode event: %v", err)
		return
	}

	log.Debugf("received event: %v", envelope)
	log.Debugf("event type: %v", envelope.EventType)

	pid := c.Param("project")
	project, err := store.GetProject(pid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "Resource Not Found"})
		log.Debugf("cannot get project ID: %v", err)
		return
	}

	log.Debugf("found project: %v", project)

	// TODO: Change this when Project.Gateways gets implemented.
	if realToken := project.Secrets["eventGridToken"]; realToken != "" {
		tok := c.Param("token")
		if realToken != tok {
			c.JSON(http.StatusForbidden, gin.H{"status": "Forbidden"})
			log.Debugf("Token does not match project's version: %v", err)
			return
		}
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		log.Debugf("failed to marshal event: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "Failed encoding"})
		return
	}

	build := &brigade.Build{
		ProjectID: pid,
		Type:      envelope.EventType,
		Provider:  "cloudevents",
		Payload:   payload,
		Revision: &brigade.Revision{
			Ref: "master",
		},
	}

	err = store.CreateBuild(build)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "Failed to invoke hook"})
		log.Debugf("failed to create build: %v", err)
		return
	}

	log.Debugf("created build: %v", build)

	// It's unclear what we are supposed to return. The spec shows a response that
	// contains the entire envelope... but it doesn't say under which conditions
	// this is to be returned. So the safest route is to return it here.
	// https://github.com/cloudevents/spec/blob/v0.1/http-transport-binding.md#324-examples
	c.JSON(http.StatusOK, envelope)
	return
}

// TODO: once the validation event is CloudEvents compliant, remove this
func validate(c *gin.Context, body io.Reader) error {
	ev, err := eventgrid.NewFromRequestBody(body)
	if err != nil {
		return err
	}

	sendValidationResponse(c, ev)
	return nil
}

// TODO: once the validation event is CloudEvents compliant, make this work with both event types
func sendValidationResponse(c *gin.Context, ev *eventgrid.Event) {
	data := ev.Data.(map[string]interface{})

	// validate endpoint - https://docs.microsoft.com/en-us/azure/event-grid/security-authentication#webhook-event-delivery
	r := gin.H{"validationResponse": data["validationCode"]}
	c.JSON(http.StatusOK, r)
	log.Debugf("sent validation response: %v", r)
}
