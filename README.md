# Newrelic Berlin 2023-02-16

This workshop is dedicated to demonstrate:

- instrumentation of applications with Open Telemetry API/SDK
- understanding the use of metrics, traces & logs
- increasing visibility in necessary parts of our codes

## Scenario

You are the mighty support engineers. You are assigned to monitor some applications which:

- are not well instrumented
- you have no direct access to
- keep making end users pissed off because of random failures

The only tool you have is New Relic where the telemetry data are being sent to. Your mission is to put an end to this misery...

## Journey

### Step 01

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
   - joe -> `FROM Metric SELECT * WHERE service.name = 'joe' AND http.client.duration IS NOT NULL SINCE 5 minutes ago`
     - is being instrumented by Open Telemetry `instrumentation.provider = opentelemetry`
     - is a Golang application `telemetry.sdk.language = go`
     - making external HTTP calls
       - with methods `http.method = GET & DELETE`
       - to `net.peer.name = donald.otel.svc.cluster.local`
       - on port `net.peer.port = 8080`
   - donald -> `FROM Metric SELECT * WHERE service.name = 'donald' AND http.server.duration IS NOT NULL SINCE 5 minutes ago`
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

### Step 02

Answers to questions 2 from step 1:

1. They are running on Kubernetes
   - `FROM Metric SELECT * WHERE service.name IN ('joe', 'donald') SINCE 5 minutes ago`
     - `k8s.node.name = otel-control-plane`
     - `k8s.namespace.name = otel`
     - `k8s.pod.name = ...`
2. Joe has 3 and donald has 2 instances
   - `FROM Metric SELECT uniques(k8s.pod.name) WHERE service.name IN ('joe', 'donald') SINCE 5 minutes ago`

Questions 1:

1. What's the average values of the golden metrics for the last 5 minutes?
2. How do they look like for different HTTP methods?
3. How do they look like for different instances?

Answers 1:

1. Golden metrics
   - donald [`server latency`]: `FROM Metric SELECT average(http.server.duration) WHERE service.name = 'donald' SINCE 5 minutes ago`
   - donald [`server throughput`]: `FROM Metric SELECT rate(count(http.server.duration), 1 minute) WHERE service.name = 'donald' SINCE 5 minutes ago`
   - donald [`server error rate`]: `FROM Metric SELECT filter(count(http.server.duration), WHERE numeric(http.status_code) >= 500)/count(http.server.duration) WHERE service.name = 'donald' SINCE 5 minutes ago`
   - joe [`client latency`]: `FROM Metric SELECT average(http.client.duration) WHERE service.name = 'joe' SINCE 5 minutes ago`
   - joe [`client throughput`]: `FROM Metric SELECT rate(count(http.client.duration), 1 minute) WHERE service.name = 'joe' SINCE 5 minutes ago`
   - joe [`client error rate`]: `FROM Metric SELECT filter(count(http.client.duration), WHERE numeric(http.status_code) >= 500)/count(http.client.duration) WHERE service.name = 'joe' SINCE 5 minutes ago`
2. Group according to HTTP methods
   - `FROM Metric SELECT average(http.client.duration) WHERE service.name = 'joe' FACET http.method SINCE 5 minutes ago`
3. Group according pods
   - `FROM Metric SELECT average(http.client.duration) WHERE service.name = 'joe' FACET k8s.pod.name SINCE 10 minutes ago`
   - `FROM Metric SELECT rate(count(http.server.duration), 1 minute) WHERE service.name = 'donald' FACET k8s.pod.name SINCE 5 minutes ago`

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
2. Can you even be sure that joe is calling donald?

Answers 2:

1. Both joe and donald have reported `500` HTTP status codes
   - `FROM Metric SELECT uniques(http.status_code) WHERE service.name = 'joe' SINCE 10 minutes ago`
   - `FROM Metric SELECT uniques(http.status_code) WHERE service.name = 'donald' SINCE 10 minutes ago`
2. 99.9%
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

### Step 03

Answers to questions 2 from step 2:

1. You know that the errors are coming from donald and reflected to joe
   - ![`step03_error_trace.png`](./docs/step03_error_trace.png)
     - joe calls donald and receives a `500`
   - You can check individual spans programmatically:
     - `FROM Span SELECT * WHERE trace.id IN (FROM Span SELECT uniques(trace.id) WHERE otel.status_code = 'ERROR') SINCE 10 minutes ago`
2. Now, you are

Questions 1:

1. What went wrong in donald?

Answers 1:

1. You don't know...

So you tell your developers to add more visibility to donald and they run the [`04_deploy_step_04.sh`](./infra/scripts/04_deploy_step_04.sh)...

### Step 04

Answers to questions 1 from step 3:

1. After getting a call from joe, donald queries a database and an error occurs:
   - ![`step04_error_trace.png`](./docs/step04_error_trace.png)
   - You can check individual spans programmatically:
     - `FROM Span SELECT * WHERE trace.id IN (FROM Span SELECT uniques(trace.id) WHERE otel.status_code = 'ERROR') SINCE 10 minutes ago`

Questions 1:

1. What can you tell about the database queries?

Answers 1:

1. Database info:
   - `db.system = mysql`
   - `db.user = root`
   - `db.name = otel`
   - `db.sql.table = names`
   - `db.operation = SELECT & DELETE`
   - `db.statement = SELECT name FROM names & DELETE FROM names`
   - `net.peer.name = mysql.otel.svc.cluster.local`
   - `net.peer.port = 3306`

**Generate some errors 😈**

First, port forward joe to localhost:

```
kubectl port-forward -n otel svc/joe 8080
```

Smash:

- `curl -X DELETE "http://localhost:8080/api?databaseConnectionError=true"`
- `curl -X GET "http://localhost:8080/api?preprocessingException=true"`
- `curl -X GET "http://localhost:8080/api?schemaNotFoundInCacheWarning=true"`

Questions 2:

1. What went wrong with the database calls?
2. What happened with joe?
3. What make donald to response so long?

You go back to your developers and say:

- We keep getting errors from our database queries
- Often joe returns `400` before it can even reach donald
  - `FROM Span SELECT * WHERE trace.id IN (FROM Span SELECT uniques(trace.id) WHERE http.status_code IS NOT NULL AND http.status_code = 400) SINCE 10 minutes ago`
- Sometimes it takes too long for donald to respond after querying the database
  - `FROM Span SELECT * WHERE service.name = 'donald' AND duration.ms > (FROM Span SELECT percentile(duration.ms, 99.9) WHERE service.name = 'donald') SINCE 10 minutes ago`

... and they run the [`05_deploy_step_05.sh`](./infra/scripts/05_deploy_step_05.sh)...

### Step 05

Answers to questions 2 from step 4:

1. You still don't know...
2. Joe seems to have some preprocessing to be done before making a call to donald.
   - It might be the following:
     - End users keep making invalid requests
       - You can improve your guidance of "how to use the platform correctly"
     - Someone with bad intentions is trying breach!
       - You need to figure out who this is and how to stop
   - `FROM Span SELECT * WHERE trace.id IN (FROM Span SELECT uniques(trace.id) WHERE service.name = 'joe' AND http.status_code = 400) SINCE 10 minutes ago`
     - `exception.type = joe.preprocessing`
     - `exception.message = Provided data format is invalid and cannot be processed.`
     - `exception.stacktrace = goroutine 143 [running]: main.performPreprocessing(0x40004e5a00, 0x4000207...`
3. Donald is making some postprocessing after retrieving some data from the database. From time to time, it takes a lot longer but you still don't know exactly why...

Apparently, metrics and traces have done their best, yet you're still lacking the root cause of some of the issues. Well, you ask for your developers to send all of the logs and they run the [`06_deploy_step_06.sh`](./infra/scripts/06_deploy_step_06.sh)...

### Step 06

**Generate some errors 😈**

First, port forward joe to localhost:

```
kubectl port-forward -n otel svc/joe 8080
```

Smash:

- `curl -X GET "http://localhost:8080/api?databaseConnectionError=true"`
- `curl -X GET "http://localhost:8080/api?tableDoesNotExistError=true"`
- `curl -X DELETE "http://localhost:8080/api?preprocessingException=true"`
- `curl -X GET "http://localhost:8080/api?schemaNotFoundInCacheWarning=true"`

Answers to questions 2 from step 4:

1. There are a couple of errors which relates to a database issue:
   - `FROM Log SELECT * WHERE level = 'error' SINCE 10 minutes ago`
     - Connection to the database is lost
     - Table does not exist
2. The log message is the same as the exception message which proves again that you were right in step 05.
3. You have found out that there is a warning log:
   - `FROM Log SELECT message WHERE level = 'warning' SINCE 10 minutes ago`
     - Message: _Processing schema not found in cache. Calculating from scratch._
       - This means that donald is using some sort of caching mechanism
       - After retrieving the data from the database, it grabs some "stuff" from the cache to perform a postprocessing
       - When the "stuff" is not in cache, it calculates whatever it needs to do from scratch...
     - You can now give more insights to your developers about how to configure caching in order to avoid this re-calculation and don't make the end users wait longer

Questions 1:

1. Can you really be sure that these logs belong to these issues?
2. Do you actually need all of these logs?

Answers 2:

1. You can't...
   - The only things you know are:
     - The correlation of the trace error or duration with the log message
     - The timestamp of the trace with errors and the timestamp of the log
     - The instance where the traces and logs are generated
   - Theoretically, those would be enough... For this dummy platform...
     - In a very complicated environment with tons of microservices, it would be _searching for needle in a haystack_
2. The `INFO` logs seem to give you a nice overview of individual steps happening in each call. But even as you were looking for answers, you haven't started with them, ain't ya?
   - So for this environment, they seem to be mostly useless and disturbing

You are one step away from perfection. You requested your developers to log only `warning` & above and you wished to attach them to your traces...

... and they run the [`07_deploy_step_07.sh`](./infra/scripts/07_deploy_step_07.sh)...

### Step 07

Answers to questions 1 from step 6:

1. Now you are sure...
   - ![`step07_error_trace_1.png`](/docs/step07_error_trace_1.png)
   - ![`step07_error_trace_2.png`](/docs/step07_error_trace_2.png)
   - You can also programmatically obtain them easily:
     - `FROM Log SELECT * WHERE trace.id IN (FROM Span SELECT uniques(trace.id) WHERE service.name = 'donald' AND duration.ms > (FROM Span SELECT percentile(duration.ms, 99.5) WHERE service.name = 'donald')) SINCE 10 minutes ago`
     - `FROM Log SELECT message WHERE trace.id IN (FROM Span SELECT uniques(trace.id) WHERE service.name = 'donald' AND otel.status_code = 'ERROR') SINCE 10 minutes ago`

Questions 1:

1. You have realized that the log messages also contain users. Can you programmatically parse that in New Relic?
2. Parsing every single time you want to query the user seems a bit exhausting? Wouldn't you want to store the user directly as a separate attribute?

Answers 1:

1. `FROM Log WITH aparse(message, 'user:*|message:*') AS (user, msg) SELECT user, msg`
2. Grok rule
   - `message LIKE '%user:%'`
   - `user:%{GREEDYDATA:user}\|message:%{GREEDYDATA:message}`
