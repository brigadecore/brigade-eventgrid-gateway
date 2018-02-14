Brigade EventGrid Gateway
=========================

[Brigade][1] gateway that responds to [Azure EventGrid][2] events.

> Please note that [EventGrid needs an HTTPS endpoint to deliver the event][3], so you need to have TLS ingress for your cluster (use something like [kube-lego][4] or [cert-manager][5])


[1]: https://github.com/azure/brigade
[2]: https://docs.microsoft.com/en-us/azure/event-grid/overview
[3]: https://docs.microsoft.com/en-us/azure/event-grid/security-authentication#webhook-event-delivery

[4]: https://github.com/jetstack/kube-lego
[5]: https://github.com/jetstack/cert-manager/