# Newrelic Berlin 2023-02-16

This workshop is dedicated to demonstrate:

- instrumentation of applications with Open Telemetry API/SDK
- understanding the use of metrics, traces & logs
- increasing visibility in necessary parts of our codes

## Scenario

You are the mighty support engineers. You are assigned to monitor some applications

- which is not well instrumented
- where you have no direct access to
- which keep making end users pissed off because of random failures

The only tool you have is New Relic where the telemetry data are being sent to. Your mission is to put an end to this misery...

## Journey

### Step 1

Your developers have already run the following scripts:

- [`00_create_kind_cluster.sh`](./infra/scripts/00_create_kind_cluster.sh)
- [`01_deploy_step_01.sh`](./infra/scripts/01_deploy_step_01.sh)

Questions 1:

1. Which services are there?
2. What telemetry data are they reporting?
3. What can you tell about these services?

Answers 1:

1. joe & donald
2. Just metrics, no traces or logs
   - `FROM Metric SELECT uniques(metricName) WHERE service.name = 'joe' SINCE 5 minutes ago`
     - `http.client.duration`
   - `FROM Metric SELECT uniques(metricName) WHERE service.name = 'donald' SINCE 5 minutes ago`
   - `http.server.duration`
   - `http.server.request_content_length`
   - `http.server.response_content_length`
3. Depending on the metrics,
   - joe -> `FROM Metric SELECT * WHERE service.name = 'joe' AND http.client.duration IS NOT NULL    SINCE 5 minutes ago`
     - is being instrumented by Open Telemetry `instrumentation.provider = opentelemetry`
     - is a Golang application `telemetry.sdk.language = go`
     - making external HTTP calls
       - with methods `http.method = GET & DELETE`
       - to `net.peer.name = donald.otel.svc.cluster.local`
       - on port `net.peer.port = 8080`
   - donald -> `FROM Metric SELECT * WHERE service.name = 'donald' AND http.server.duration IS NOT    NULL SINCE 5 minutes ago`
     - is being instrumented by Open Telemetry `instrumentation.provider = opentelemetry`
     - is a Golang application `telemetry.sdk.language = go`
     - is an HTTP server
       - on host `net.host.name = donald.otel.svc.cluster.local`
       - with port `net.host.port = 8080`
       - receiving calls with methods `http.method = GET & DELETE`

Questions 2:

1. Can you tell where these applications are running?
2. Can you tell how many instances each service has?

Answers 2:

- Nope

So you ask your fellow developers to add some metadata to their applications and they run the [`02_deploy_step_02.sh`](./infra/scripts/02_deploy_step_02.sh)...

### Step 2

Answers to questions 2 from step 1:

1. They are running on Kubernetes
   - `FROM Metric SELECT * WHERE service.name IN ('joe', 'donald') SINCE 5 minutes ago`
     - `k8s.node.name = otel-control-plane`
     - `k8s.namespace.name = otel`
     - `k8s.pod.name = ...`
2. Joe has 2 and donald has 3 instances
   - `FROM Metric SELECT uniqueCount(k8s.pod.name) WHERE service.name IN ('joe', 'donald') FACET service.name SINCE 5 minutes ago`

Questions 1:

1. What's the average values of the golden metrics for the last 10 minutes?
2. How do they look like for different HTTP methods?
3. How do they look like for different instances?

Answers 1:

1. Golden metrics
   - donald [`server latency`]: `FROM Metric SELECT average(http.server.duration) WHERE service.name = 'donald' SINCE 10 minutes ago`
   - donald [`server throughput`]: `FROM Metric SELECT rate(count(http.server.duration), 1 minute) WHERE service.name = 'donald' SINCE 10 minutes ago`
   - donald [`server throughput`]: `FROM Metric SELECT filter(count(%.server.duration), WHERE numeric(http.status_code) >= 500)/count(%.server.duration) WHERE service.name = 'donald' SINCE 10 minutes ago`
   - joe [`client latency`]: `FROM Metric SELECT average(http.client.duration) WHERE service.name = 'joe' SINCE 10 minutes ago`
   - joe [`client throughput`]: `FROM Metric SELECT rate(count(http.client.duration), 1 minute) WHERE service.name = 'joe' SINCE 10 minutes ago`
   - joe [`client throughput`]: `FROM Metric SELECT filter(count(%.client.duration), WHERE numeric(http.status_code) >= 500)/count(%.client.duration) WHERE service.name = 'joe' SINCE 10 minutes ago`
2. Group according to HTTP methods
   - `FROM Metric SELECT average(http.client.duration) WHERE service.name = 'joe' FACET http.method SINCE 10 minutes ago`
3. Group according pods
   - `FROM Metric SELECT average(http.client.duration) WHERE service.name = 'joe' FACET k8s.pod.name SINCE 10 minutes ago`

**Generate some errors 😈**

First, port forward joe to localhost:

```
kubectl port-forward -n otel svc/joe 8080
```

Smash:

- `curl -X GET "http://localhost:8080/api?databaseConnectionError=true"`
- `curl -X GET "http://localhost:8080/api?tableDoesNotExistError=true"`

Questions 2:

1. Where are the errors coming from?
2. What is the cause for these errors?
3. Can you even be sure that joe is calling donald?

Answers 2:

1. Both joe and donald have reported `500` HTTP status codes
   - `FROM Metric SELECT uniques(http.status_code) WHERE service.name = 'joe' SINCE 10 minutes ago`
   - `FROM Metric SELECT uniques(http.status_code) WHERE service.name = 'donald' SINCE 10 minutes ago`
2. You don't know...
3. 99.9%
   - Metric metadata:
     - `FROM Metric SELECT * WHERE service.name = 'joe' SINCE 10 minutes ago`
     - `FROM Metric SELECT * WHERE service.name = 'donald' SINCE 10 minutes ago`
   - joe is making HTTP calls:
     - to `net.peer.name = donald.otel.svc.cluster.local`
     - on port `net.peer.port = 8080`
   - donald is accepting HTTP calls:
     - as `net.host.name = donald.otel.svc.cluster.local`
     - on port `net.host.port = 8080`
   - joe and donald are running on the same node & namespace
     - `k8s.node.name = otel-control-plane`
     - `k8s.namespace.name = otel`
   - Since the calls are cluster internal (`...svc.cluster.local`), the peer of joe must be host of donald
   - Yet, you don't know whether the services are on the same cluster!
     - `k8s.cluster.name = ???`

So you tell your developers to introduce some traces to their applications and they run the [`03_deploy_step_03.sh`](./infra/scripts/03_deploy_step_03.sh)...

### Step 3

Answers to questions 2 from step 2:

1. You know that the errors are coming from donald and reflected to joe
   - `FROM Span SELECT * WHERE service.name = 'joe' AND otel.status_code = 'ERROR' SINCE 10 minutes ago`
   - `FROM Span SELECT * WHERE service.name = 'donald' AND otel.status_code = 'ERROR' SINCE 10 minutes ago`
2. You still don't know...

### Step 4

Answers to questions 2 from step 2:

## Wrap up
