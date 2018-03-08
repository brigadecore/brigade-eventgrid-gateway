Brigade EventGrid Gateway
=========================

[Brigade][1] gateway that responds to [Azure EventGrid][2] events.

> Please note that [EventGrid needs an HTTPS endpoint to deliver the event][3], so you need to have TLS ingress for your cluster (use something like [kube-lego][4] or [cert-manager][5]) - this chart assumes you have an nginx ingress controller deployed on your cluster.


First you need to clone this repo: 

`git clone https://github.com/radu-matei/brigade-eventgrid-gateway` and navigate to the root directory.

Deploying the gateway
---------------------

> [Here you can read more about Brigade Gateways][6]

`helm install -n brigade-eventgrid-gateway ./charts/brigade-eventgrid-gateway --set ingress.host=<your-HTTPS-endpoint>`

> You can also  specify the host in [`charts/brigade-eventgrid-gateway/values.yaml`][7]

At this point, you should be able to navigate to `https://<your-endpoint>/healthz` and receive `"message": "ok"`. 

Creating the Azure EventGrid subscription
-----------------------------------------

You have two options to setup your endpoint as a receiver for Azure EventGrid events: manually, throught the Azure Portal, or programatically, through the other chart from this repo.

> Before registering the endpoint, you should also have a brigade project configured - [check the Brigade Quickstart][8], and learn how to use the `brig` command line - you will use it to retrieve the project ID, and an Azure Storage Account.

> At the moment we will only subscribe to blob storage events

Options 1: Using the Helm chart
--------------------------------

To execute commands against your Azure subscription, you need to use your Azure credentials (service principal - [here's how to create a new one from the portal][9]).

Use the values from the servive principal and populate this Kubernetes secret (with values in base 64) (and **do not commit it to source control!**):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: eventgrid-subscriber-secret
type: Opaque
data:
  AZ_SUBSCRIPTION_ID: <azure-subscription-id-base64>
  AZ_TENANT_ID: <azure-tenant-id-base64>
  AZ_CLIENT_ID: <azure-client-id-base64>
  AZ_CLIENT_SECRET: <azure-client-secret-base64>
```

Modify the [values from `/charts/eventgrid-subscriber/values.yaml`][10], with `WEBHOOK_URL` being `https://<your-endpoint>/eventgrid/<brigade-project-id>`, then `helm install -n subscriber ./charts/eventgrid-subscriber` - this will create a Kubernetes job which, if the credentials and values are correct, will create an event subscription and send any storage events from that storage account back to the URL you specified - which is the gateway we deoployed earlier.


Option 2: Using the Azure Portal
--------------------------------

You can [follow this tutorial][11] and create an event subscription on the same URL as before - essentially, this is exactly the same operation as above, only done manually in the Azure Portal.


If you generate blob storage events and check the logs of the gateway, you can see that the gateway is creating Brigade builds - now you need to create a `brigade.js` and handle these events!

More documentation and examples soon... :)


[1]: https://github.com/azure/brigade
[2]: https://docs.microsoft.com/en-us/azure/event-grid/overview
[3]: https://docs.microsoft.com/en-us/azure/event-grid/security-authentication#webhook-event-delivery

[4]: https://github.com/jetstack/kube-lego
[5]: https://github.com/jetstack/cert-manager/

[6]: https://github.com/Azure/brigade/blob/master/docs/topics/gateways.md
[7]: charts/brigade-eventgrid-gateway/values.yaml

[8]: https://github.com/Azure/brigade#quickstart
[9]: https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal
[10]: /charts/eventgrid-subscriber/values.yaml

[11]: https://docs.microsoft.com/en-us/azure/event-grid/custom-event-quickstart-portal#subscribe-to-a-topic