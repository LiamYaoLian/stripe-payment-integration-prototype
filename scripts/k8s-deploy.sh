#!/usr/bin/env bash
# Ordered Kubernetes rollout for stripe-payment-api.
# Prerequisites:
#   - kubectl context set to target cluster
#   - deploy/kubernetes/configmap.yaml (from configmap.example.yaml)
#   - deploy/kubernetes/secrets.yaml (from secrets.example.yaml)
#   - migrations ConfigMap: kubectl create configmap stripe-payment-migrations \
#       --from-file=backend/migrations --dry-run=client -o yaml | kubectl apply -f -
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
K8S="${ROOT}/deploy/kubernetes"

echo "Applying ConfigMap and Secrets..."
kubectl apply -f "${K8S}/configmap.yaml"
kubectl apply -f "${K8S}/secrets.yaml"

echo "Refreshing migrations ConfigMap..."
kubectl create configmap stripe-payment-migrations \
  --from-file="${ROOT}/backend/migrations" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "Running migrate job..."
kubectl delete job stripe-payment-migrate --ignore-not-found
kubectl apply -f "${K8S}/migrate-job.yaml"
kubectl wait --for=condition=complete job/stripe-payment-migrate --timeout=120s

echo "Rolling out API..."
kubectl apply -f "${K8S}/service.yaml"
kubectl apply -f "${K8S}/api-deployment.yaml"
kubectl apply -f "${K8S}/pdb.yaml"
kubectl rollout status deployment/stripe-payment-api --timeout=180s

echo "Rolling out web..."
kubectl apply -f "${K8S}/web-service.yaml"
kubectl apply -f "${K8S}/web-deployment.yaml"
kubectl rollout status deployment/stripe-payment-web --timeout=180s

if [[ -f "${K8S}/ingress.yaml" ]]; then
  echo "Applying Ingress..."
  kubectl apply -f "${K8S}/ingress.yaml"
else
  echo "No ingress.yaml — copy ingress.example.yaml when ready."
fi

echo "Optional observability manifests (Prometheus Operator):"
echo "  kubectl apply -f deploy/prometheus/servicemonitor.yaml"
echo "  kubectl apply -f deploy/prometheus/prometheusrule.yaml"
echo "Deploy complete."
