#!/bin/sh

# Microk8s
DOCKER_REGISTRY=localhost:32000

docker build . -t $DOCKER_REGISTRY/keptn-dt-synthetic-service:dev
docker push $DOCKER_REGISTRY/keptn-dt-synthetic-service:dev

# Pull new image by deleting current pod
kubectl -n keptn delete pod -l app.kubernetes.io/instance=keptn-dt-synthetic-service

# helm upgrade --install -n keptn keptn-dt-synthetic-service chart/ --set image.repository=localhost:32000/keptn-dt-synthetic-service --set image.tag=dev
# kubectl -n keptn logs -l app.kubernetes.io/instance=keptn-dt-synthetic-service -c keptn-service --follow
