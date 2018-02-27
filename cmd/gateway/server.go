package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"

	"github.com/Azure/brigade/pkg/brigade"
	"github.com/Azure/brigade/pkg/storage"
	"github.com/Azure/brigade/pkg/storage/kube"
	eventgrid "github.com/radu-matei/brigade-eventgrid-gateway/pkg/eventgrid"

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

	router.Run()
}

func azFn(c *gin.Context) {

	defer c.Request.Body.Close()
	decoder := json.NewDecoder(c.Request.Body)
	var received []eventgrid.Event

	err := decoder.Decode(&received)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Malformed body"})
		log.Debugf("cannot decode event: %v", err)
	}
	ev := received[0]
	log.Debugf("received event: %v", ev)

	if ev.EventType == eventgrid.ValidationEvent {
		data := ev.Data.(map[string]interface{})

		// validate endpoint - https://docs.microsoft.com/en-us/azure/event-grid/security-authentication#webhook-event-delivery
		r := gin.H{"validationCode": data["validationCode"]}
		c.JSON(http.StatusOK, r)
		log.Debugf("sent validation response: %v", r)
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

	payload, err := json.Marshal(ev)
	if err != nil {
		log.Debugf("failed to marshal event: %v", err)
	}

	build := &brigade.Build{
		ProjectID: pid,
		Type:      ev.EventType,
		Provider:  "eventgrid",
		Commit:    "master",
		Payload:   payload,
	}

	err = store.CreateBuild(build)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "Failed to invoke hook"})
		log.Debugf("failed to create build: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OK"})
	return
}
