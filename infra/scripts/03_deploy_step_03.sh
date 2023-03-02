#!/bin/bash

###################
### Parse input ###
###################

while (( "$#" )); do
  case "$1" in
    --platform)
      platform="$2"
      shift
      ;;
    --build)
      build="true"
      shift
      ;;
    *)
      shift
      ;;
  esac
done

# Docker platform
if [[ $platform == "" ]]; then
  # Default is amd
  platform="amd64"
else
  if [[ $platform != "amd64" && $platform != "arm64" ]]; then
    echo "Platform can either be 'amd64' or 'arm64'."
    exit 1
  fi
fi

#####################
### Set variables ###
#####################

repoName="newrelic-berlin-2023-02-16"

# mysql
declare -A mysql
mysql["name"]="mysql"
mysql["namespace"]="otel"
mysql["username"]="root"
mysql["password"]="verysecretpassword"
mysql["port"]=3306
mysql["database"]="otel"
mysql["table"]="names"

# otelcollector
declare -A otelcollector
otelcollector["name"]="otel-collector"
otelcollector["namespace"]="otel"
otelcollector["mode"]="deployment"

# donald
declare -A donald
donald["name"]="donald"
donald["imageName"]="${repoName}:${donald[name]}-${platform}"
donald["namespace"]="otel"
donald["replicas"]=1
donald["port"]=8080

# joe
declare -A joe
joe["name"]="joe"
joe["imageName"]="${repoName}:${joe[name]}-${platform}"
joe["namespace"]="otel"
joe["replicas"]=1
joe["port"]=8080
joe["interval"]=2000

####################
### Build & Push ###
####################

if [[ $build == "true" ]]; then
  # donald
  docker build \
    --platform "linux/${platform}" \
    --tag "${DOCKERHUB_NAME}/${donald[imageName]}" \
    "../../apps/${donald[name]}/."
  docker push "${DOCKERHUB_NAME}/${donald[imageName]}"

  # joe
  docker build \
    --platform "linux/${platform}" \
    --tag "${DOCKERHUB_NAME}/${joe[imageName]}" \
    "../../apps/${joe[name]}/."
  docker push "${DOCKERHUB_NAME}/${joe[imageName]}"
fi

###################
### Deploy Helm ###
###################

# Add helm repos
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# mysql
helm upgrade ${mysql[name]} \
  --install \
  --wait \
  --debug \
  --create-namespace \
  --namespace=${mysql[namespace]} \
  --set auth.rootPassword=${mysql[password]} \
  --set auth.database=${mysql[database]} \
    "bitnami/mysql"

# otelcollector
helm upgrade ${otelcollector[name]} \
  --install \
  --wait \
  --debug \
  --create-namespace \
  --namespace ${otelcollector[namespace]} \
  --set mode=${otelcollector[mode]} \
  --set presets.kubernetesAttributes.enabled=true \
  --set serviceAccount.create=true \
  --set config.receivers.jaeger=null \
  --set config.receivers.prometheus=null \
  --set config.receivers.zipkin=null \
  --set config.processors.cumulativetodelta.include.match_type="strict" \
  --set config.processors.cumulativetodelta.include.metrics[0]="http.server.duration" \
  --set config.processors.cumulativetodelta.include.metrics[1]="http.client.duration" \
  --set config.processors.k8sattributes.passthrough=false \
  --set config.processors.k8sattributes.extract.metadata[0]="k8s.cluster.name" \
  --set config.processors.k8sattributes.extract.metadata[1]="k8s.node.name" \
  --set config.processors.k8sattributes.extract.metadata[2]="k8s.namespace.name" \
  --set config.processors.k8sattributes.extract.metadata[3]="k8s.pod.name" \
  --set config.exporters.otlp.endpoint="otlp.eu01.nr-data.net:4317" \
  --set config.exporters.otlp.tls.insecure=false \
  --set config.exporters.otlp.headers.api-key=$NEWRELIC_LICENSE_KEY \
  --set config.service.pipelines.traces.receivers[0]="otlp" \
  --set config.service.pipelines.traces.processors[0]="batch" \
  --set config.service.pipelines.traces.processors[1]="memory_limiter" \
  --set config.service.pipelines.traces.exporters[0]="otlp" \
  --set config.service.pipelines.metrics.receivers[0]="otlp" \
  --set config.service.pipelines.metrics.processors[0]="batch" \
  --set config.service.pipelines.metrics.processors[1]="memory_limiter" \
  --set config.service.pipelines.metrics.processors[2]="cumulativetodelta" \
  --set config.service.pipelines.metrics.exporters[0]="otlp" \
  --set config.service.pipelines.logs=null \
  "open-telemetry/opentelemetry-collector"

# donald
helm upgrade ${donald[name]} \
  --install \
  --wait \
  --debug \
  --create-namespace \
  --namespace=${donald[namespace]} \
  --set dockerhubName=$DOCKERHUB_NAME \
  --set imageName=${donald[imageName]} \
  --set imagePullPolicy="Always" \
  --set name=${donald[name]} \
  --set replicas=${donald[replicas]} \
  --set port=${donald[port]} \
  --set mysql.server="${mysql[name]}.${mysql[namespace]}.svc.cluster.local" \
  --set mysql.username=${mysql[username]} \
  --set mysql.password=${mysql[password]} \
  --set mysql.port=${mysql[port]} \
  --set mysql.database=${mysql[database]} \
  --set mysql.table=${mysql[table]} \
  --set otlp.endpoint="http://${otelcollector[name]}-opentelemetry-collector.${otelcollector[namespace]}.svc.cluster.local:4317" \
  --set features.considerDatabaseSpans="false" \
  "../helm/${donald[name]}"

# joe
helm upgrade ${joe[name]} \
  --install \
  --wait \
  --debug \
  --create-namespace \
  --namespace=${joe[namespace]} \
  --set dockerhubName=$DOCKERHUB_NAME \
  --set imageName=${joe[imageName]} \
  --set imagePullPolicy="Always" \
  --set name=${joe[name]} \
  --set replicas=${joe[replicas]} \
  --set port=${joe[port]} \
  --set donald.requestInterval=${joe[interval]} \
  --set donald.endpoint="${donald[name]}.${donald[namespace]}.svc.cluster.local" \
  --set donald.port="${donald[port]}" \
  --set otlp.endpoint="http://${otelcollector[name]}-opentelemetry-collector.${otelcollector[namespace]}.svc.cluster.local:4317" \
  "../helm/${joe[name]}"
