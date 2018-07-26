# Brigade EventGrid Gateway

[Brigade][1] gateway that responds to [Azure EventGrid][2] events.

> [Here you can read more about Brigade Gateways][6]

Deploying the gateway
---------------------

First you need to clone this repo: 

`git clone https://github.com/radu-matei/brigade-eventgrid-gateway` and navigate to the root directory.

[EventGrid needs an HTTPS endpoint to deliver the event][3], so you need to have TLS ingress for your cluster (use something like [kube-lego][4] or [cert-manager][5]), and this chart assumes you have an `nginx` ingress controller deployed on your cluster. Once you have an HTTPS domain or subdomain, pass it to the `ingress.host` property below.

Now install the Helm chart:

`helm install -n brigade-eventgrid-gateway ./charts/brigade-eventgrid-gateway --set ingress.host=<your-HTTPS-endpoint>`

> You can also  specify the host in [`charts/brigade-eventgrid-gateway/values.yaml`][7]

> By default, the chart assumes you have a cluster with RBAC enabled. If you don't, either modify the `rbac.enabled` value in `values.yaml` or pass `--set rbac.enabled=false` to the `helm install` command. 

> If you don't have a domain or an ingress controller configured, you can [change the ingress annotations in the ingress template file](charts/brigade-eventgrid-gateway/templates/ingress.yaml) - but keep in mind that EventGrid will not pass events to a non-HTTPS endpoint.

At this point, you should be able to navigate to `https://<your-endpoint>/healthz` and receive `"message": "ok"` and you can start sending events to this gateway.


## Creating a Brigade project

You can [follow the instructions from the official Brigade documentation](https://github.com/Azure/brigade/blob/master/docs/topics/projects.md) to create a new project - the gateway will use a token in order to make sure that if unauthorized people send events to your gateway, those will not become Brigade builds - the token is passed in the URL and it is checked whenever a new event is received, before creating a new Brigade build, and can be any string. In your project's `values.yaml` file, add an `eventGridToken` token:

```
project: "<your-project>"
repository: "<your-repo>"
cloneURL: "<your-clone-url>"

secrets:
  eventGridToken: "<your-token>"

allowPrivilegedJobs: "false"
```


Then create the project:

`helm install -n eventgrid-project brigade/brigade-project -f values.yaml`

When creating an EventGrid subscription you will need both the project ID and the token, as they will be part of the event endpoint URL.

## Creating the Azure EventGrid subscription

Azure EventGrid supports two JSON event schemas:

- [the CloudEvents JSON schema](https://github.com/cloudevents/spec/blob/master/json-format.md), which is an open standard for describing event data in a consistent way (which is in preview at the moment of writing this document)
- [the default Azure EventGrid schema](https://docs.microsoft.com/en-us/azure/event-grid/event-schema)

For the purpose of this tutorial we will use a storage account as the source of events, but the same concepts can be applied to [any event source Azure EventGrid supports](https://docs.microsoft.com/en-us/azure/event-grid/overview):

```
az storage account create \
  --name  <storage-account-name> \
  --location northeurope \
  --resource-group <resource-group-name> \
  --sku Standard_LRS \
  --kind BlobStorage \
  --access-tier Hot
```

> Please note that currently, Azure Event Grid has preview support for CloudEvents JSON format input and output in West Central US, Central US, and North Europe.

> For more up-to-date information about regions and CloudEvents in Azure EventGrid, [check out the documentation][13]

In order to create an event subscription, we need to pass the id to the resource that generates the events - in this case, we need to pass the id to the storage account we just created:

`storageid=$(az storage account show --name <storage-account-name> --resource-group <resource-group-name> --query id --output tsv)`

> Please that in this way you can get the ID of any Azure resource that supports generating events and sending them to EventGrid.
> For an up-to-date list of Azure resources that support EventGrid, [check out the documentation][12]

### Using the CloudEvents schema

We want to generate events from Azure resources using the CloudEvents schema and handle them using the gateway we just deployed. Since the feature is currently in preview, we need to add an extension for the `az` command line:

```
az extension add --name eventgrid
```

Then, we create the event subscription:

```
  az eventgrid event-subscription create \
  --resource-id $storageid \
  --name brigade-cloudevents \
  --endpoint https://<your-endpoint>/cloudevents/v0.1/<brigade-project-id>/<your-token> \
  --event-delivery-schema cloudeventv01schema
```

> You can get the Brigade Project ID using the [`brig`][11] tool: `brig project list`

Note that the path for CloudEvents is `/cloudevents/v0.1/<brigade-project-id>/<your-token>`, and an example of the full URL would be:

`https://$YOURDOMAIN/cloudevents/v0.1/brigade-9e0af40182d1ab201542ebfb7d795d189ad0ec0b73512190e93935/you-should-change-this-to-be-your-own`

This is a sample event that follows the CloudEvents schema looks like:

```
{
    "cloudEventsVersion" : "0.1",
    "eventType" : "Microsoft.Storage.BlobCreated",
    "eventTypeVersion" : "",
    "source" : "/subscriptions/{subscription-id}/resourceGroups/{resource-group}/providers/Microsoft.Storage/storageAccounts/{storage-account}#blobServices/default/containers/{storage-container}/blobs/{new-file}",
    "eventID" : "173d9985-401e-0075-2497-de268c06ff25",
    "eventTime" : "2018-04-28T02:18:47.1281675Z",
    "data" : {
      "api": "PutBlockList",
      "clientRequestId": "6d79dbfb-0e37-4fc4-981f-442c9ca65760",
      "requestId": "831e1650-001e-001b-66ab-eeb76e000000",
      "eTag": "0x8D4BCC2E4835CD0",
      "contentType": "application/octet-stream",
      "contentLength": 524288,
      "blobType": "BlockBlob",
      "url": "https://oc2d2817345i60006.blob.core.windows.net/oc2d2817345i200097container/oc2d2817345i20002296blob",
      "sequencer": "00000000000004420000000000028963",
      "storageDiagnostics": {
        "batchId": "b68529f3-68cd-4744-baa4-3c0498ec19f0"
      }
    }
}
```

### Using the default EventGrid schema

```
  az eventgrid event-subscription create \
  --resource-id $storageid \
  --name brigade-eventgrid \
  --endpoint https://<your-endpoint>/eventgrid/<brigade-project-id>/<your-token> 
```

Note that the path for the default EventGrid schema is `/eventgrid/<brigade-project-id>/<your-token>`

This is a sample event that follows the Azure EventGrid default schema:

```
[{
  "topic": "/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourceGroups/myrg/providers/Microsoft.Storage/storageAccounts/myblobstorageaccount",
  "subject": "/blobServices/default/containers/testcontainer/blobs/testfile.txt",
  "eventType": "Microsoft.Storage.BlobCreated",
  "eventTime": "2017-08-16T20:33:51.0595757Z",
  "id": "4d96b1d4-0001-00b3-58ce-16568c064fab",
  "data": {
    "api": "PutBlockList",
    "clientRequestId": "d65ca2e2-a168-4155-b7a4-2c925c18902f",
    "requestId": "4d96b1d4-0001-00b3-58ce-16568c000000",
    "eTag": "0x8D4E4E61AE038AD",
    "contentType": "text/plain",
    "contentLength": 0,
    "blobType": "BlockBlob",
    "url": "https://myblobstorageaccount.blob.core.windows.net/testcontainer/testblob1.txt",
    "sequencer": "00000000000000EB0000000000046199",
    "storageDiagnostics": {
      "batchId": "dffea416-b46e-4613-ac19-0371c0c5e352"
    }
  },
  "dataVersion": "",
  "metadataVersion": "1"
}]
```

> You can get the Brigade Project ID using the [`brig`][11] tool: `brig project list`

An example of the full URL would be:

`https://$YOURDOMAIN/eventgrid/brigade-9e0af40182d1ab201542ebfb7d795d189ad0ec0b73512190e93935/you-should-change-this-to-be-your-own`

In both cases, a validation request will be sent to the endpoint, which this gateway handles - after this, the endpoint will receive events according to the subscription.


## Handling events in Brigade builds

Following the example so far, blob storage generates two events: `Microsoft.Storage.BlobCreated` and `Microsoft.Storage.BlobDeleted` that we can handle in our `brigade.js` file:

```javascript
const { events } = require('brigadier')

events.on("Microsoft.Storage.BlobDeleted", (e, p) => {
  console.log(e);
})

events.on("Microsoft.Storage.BlobCreated", (e, p) => {
  console.log(e);
})
```

At this point, whenever events are fired, our simple handlers will log them to the console:

```shell
$ brig build logs 01cegwv9t48kva8wh093pw0hbn

==========[  brigade-worker-01cegwv9t48kva8wh093pw0hbn  ]==========
prestart: empty script found. Falling back to VCS script
prestart: src/brigade.js written[brigade] brigade-worker version: 0.14.0
[brigade:k8s] Creating PVC named brigade-worker-01cegwv9t48kva8wh093pw0hbn
{ buildID: '01cegwv9t48kva8wh093pw0hbn',
  workerID: 'brigade-worker-01cegwv9t48kva8wh093pw0hbn',
  type: 'Microsoft.Storage.BlobDeleted',
  provider: 'cloudevents',
  revision: { commit: '', ref: 'master' },
  logLevel: 1,
  payload: '{"eventType":"Microsoft.Storage.BlobDeleted","eventTypeVersion":"","cloudEventsVersion":"0.1","source":"/subscriptions/<subscription-id>/resourceGroups/<resource-group>/providers/Microsoft.Storage/storageAccounts/<storage-account>#blobServices/default/containers/<path-to-file-in-blob>","eventID":"<event-id>","eventTime":"2018-05-27T13:33:18.1443969Z","contentType":"","extensions":null,"data":{"api":"DeleteBlob","blobType":"BlockBlob","contentLength":5698,"contentType":"application/octet-stream","eTag":"<e-tag>","requestId":"<request-id>","sequencer":"<sequencer>","storageDiagnostics":{"batchId":"<batch-id>"},"url":"https://<storage-account>.blob.core.windows.net/<path-to-file-in-blob>"}}' }
[brigade:app] after: default event handler fired
[brigade:app] beforeExit(2): destroying storage
[brigade:k8s] Destroying PVC named brigade-worker-01cegwv9t48kva8wh093pw0hbn
```

# Building from source and running locally

Prerequisites:
- [the Go toolchain][8]
- [`dep`][9]
- `make` (optional)

To build from source:

- `dep ensure`
- `make build` or `go build` to build the binary for your OS
- if running locally, you should provide an environment variable for the Kubernetes configuration file:
  - on Linux (including Windows Subsystem for Linux) and macOS: `export KUBECONFIG=<path-to-config>`
  - on Windows: `$env:KUBECONFIG="<path-to-config>"` 

- starting the binary you should see the initial Gin output:

```
[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /healthz                  --> main.healthz (2 handlers)
[GIN-debug] POST   /eventgrid/:project       --> main.azFn (3 handlers)
[GIN-debug] POST   /eventgrid/:project/:token --> main.azFn (3 handlers)
[GIN-debug] POST   /cloudevents/v0.1/:project/:token --> main.ceFn (3 handlers)
[GIN-debug] Environment variable PORT is undefined. Using port :8080 by default
[GIN-debug] Listening and serving HTTP on :8080
```
- at this point, your server should be able to start accepting incoming requests to `localhost:8080`
- you can test the server locally, using [Postman][10] (POST requests with your desired JSON payload - see the `testdata` folders used for testing)
- please note that running locally with a Kubernetes config file set is equivalent to running privileged inside the cluster, and any Brigade builds created will get executed!


[1]: https://github.com/azure/brigade
[2]: https://docs.microsoft.com/en-us/azure/event-grid/overview
[3]: https://docs.microsoft.com/en-us/azure/event-grid/security-authentication#webhook-event-delivery

[4]: https://github.com/jetstack/kube-lego
[5]: https://github.com/jetstack/cert-manager/

[6]: https://github.com/Azure/brigade/blob/master/docs/topics/gateways.md
[7]: charts/brigade-eventgrid-gateway/values.yaml

[8]: https://golang.org/doc/install
[9]: https://github.com/golang/dep

[10]: https://www.getpostman.com/
[11]: https://github.com/Azure/brigade/releases

[12]: https://docs.microsoft.com/en-us/azure/event-grid/overview
[13]: https://docs.microsoft.com/en-us/azure/event-grid/cloudevents-schema
