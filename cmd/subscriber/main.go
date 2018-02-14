package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/eventgrid/mgmt/2018-01-01/eventgrid"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
)

var (
	defaultActiveDirectoryEndpoint = azure.PublicCloud.ActiveDirectoryEndpoint
	defaultResourceManagerEndpoint = azure.PublicCloud.ResourceManagerEndpoint

	subscriptionID = getEnvVarOrExit("AZ_SUBSCRIPTION_ID")
	tenantID       = getEnvVarOrExit("AZ_TENANT_ID")
	clientID       = getEnvVarOrExit("AZ_CLIENT_ID")
	clientSecret   = getEnvVarOrExit("AZ_CLIENT_SECRET")

	resourceGroup  = getEnvVarOrExit("RESOURCE_GROUP")
	storageAccount = getEnvVarOrExit("STORAGE_ACCOUNT")
	name           = getEnvVarOrExit("EVENTGRID_SUBSCRIPTION_NAME")
	webhookURL     = getEnvVarOrExit("WEBHOOK_URL")
)

func main() {
	c, err := getEventGridClient(subscriptionID, tenantID, clientID, clientSecret)
	if err != nil {
		log.Fatalf("cannot get eventgrid client: %v", err)
	}

	scope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/microsoft.storage/storageAccounts/%s", subscriptionID, resourceGroup, storageAccount)
	ctx := context.Background()

	subscription := eventgrid.EventSubscription{
		EventSubscriptionProperties: &eventgrid.EventSubscriptionProperties{
			Destination: eventgrid.WebHookEventSubscriptionDestination{
				EndpointType: eventgrid.EndpointTypeWebHook,
				WebHookEventSubscriptionDestinationProperties: &eventgrid.WebHookEventSubscriptionDestinationProperties{
					EndpointURL: to.StringPtr(webhookURL),
				},
			},
		},
	}

	f, err := c.CreateOrUpdate(ctx, scope, name, subscription)
	if err != nil {
		log.Fatalf("cannot create event subscription: %v", err)
	}

	err = f.WaitForCompletion(ctx, c.Client)
	if err != nil {
		log.Fatalf("cannot get the subscription create or update future response: %v", err)
	}

	log.Printf("eventgrid subscription %s now has URL %s", name, webhookURL)
}

func getEventGridClient(subscriptionID, tenantID, clientID, clientSecret string) (eventgrid.EventSubscriptionsClient, error) {
	var subscriptionsClient eventgrid.EventSubscriptionsClient

	oAuthConfig, err := adal.NewOAuthConfig(defaultActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return subscriptionsClient, fmt.Errorf("cannot get oauth config: %v", err)
	}
	token, err := adal.NewServicePrincipalToken(*oAuthConfig, clientID, clientSecret, defaultResourceManagerEndpoint)
	if err != nil {
		return subscriptionsClient, fmt.Errorf("cannot get service principal token: %v", err)
	}

	subscriptionsClient = eventgrid.NewEventSubscriptionsClient(subscriptionID)
	subscriptionsClient.Authorizer = autorest.NewBearerAuthorizer(token)

	return subscriptionsClient, nil
}
func getEnvVarOrExit(varName string) string {
	value := os.Getenv(varName)
	if value == "" {
		log.Fatalf("missing environment variable %s\n", varName)
	}

	return value
}
