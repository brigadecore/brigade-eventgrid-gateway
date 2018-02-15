package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/brigade/pkg/storage"

	eventgrid "github.com/radu-matei/brigade-eventgrid-gateway/pkg/eventgrid"

	"github.com/gin-gonic/gin"
)

var store storage.Store

func main() {

	// client, err := kube.GetClient("", os.Getenv("KUBECONFIG"))
	// if err != nil {
	// 	panic(err)
	// }
	// store = kube.New(client, "default")

	//router := gin.Default()
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "ok"}) })

	router.POST("/", azFn)

	router.Run()
}

func azFn(c *gin.Context) {
	defer c.Request.Body.Close()
	decoder := json.NewDecoder(c.Request.Body)
	var ev eventgrid.Event

	err := decoder.Decode(&ev)
	if err != nil {
		panic(err)
	}

	if ev.EventType == "Microsoft.EventGrid.SubscriptionValidationEvent" {
		fmt.Printf("%v", ev.Data)
		switch data := ev.Data.(type) {
		case eventgrid.DataValidation:
			fmt.Printf("%v", data)

		default:
			fmt.Printf("I don't know...")
		}
	}

}

// func trelloFn(c *gin.Context) {
// 	pid := c.Param("project")

// 	// INCOMPLETE: This is the first step in validating the request originated
// 	// from Trello. Finish. https://developers.trello.com/page/webhooks
// 	sig := c.Request.Header.Get("x-trello-webhook")
// 	if sig == "" {
// 		log.Println("No X-Trello-Webhook header present. Skipping")
// 		c.JSON(http.StatusBadRequest, gin.H{"status": "Malformed headers"})
// 		return
// 	}
// 	// TODO: validate that the body matches the hash in the sig header.

// 	body, err := ioutil.ReadAll(c.Request.Body)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"status": "Malformed body"})
// 		return
// 	}
// 	c.Request.Body.Close()

// 	// Load project
// 	proj, err := store.GetProject(pid)
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"status": "Resource Not Found"})
// 		return
// 	}

// 	// Create the build
// 	build := &brigade.Build{
// 		ProjectID: pid,
// 		Type:      "trello",
// 		Provider:  "trello",
// 		Commit:    "master",
// 		Payload:   body,
// 	}

// 	if err := store.CreateBuild(build); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"status": "Failed to invoke hook"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "OK"})
// }
