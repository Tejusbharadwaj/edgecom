#!/bin/bash

# =================================================================
# Kubernetes Cleanup Script
# =================================================================
# WARNING: This script is ONLY for Kubernetes deployment cleanup.
# If you're using Docker Compose, use: docker compose down
# =================================================================

cd "$(dirname "$0")/.." || exit

# Remove all Kubernetes resources
kubectl delete -f k8s/service.yaml
kubectl delete -f k8s/deployment.yaml
kubectl delete -f k8s/database-service.yaml
kubectl delete -f k8s/database.yaml
kubectl delete -f k8s/config.yaml

echo "Kubernetes resources have been cleaned up" 