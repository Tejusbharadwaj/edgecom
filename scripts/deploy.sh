#!/bin/bash

# =================================================================
# Kubernetes Deployment Script
# =================================================================
# WARNING: This script is ONLY for Kubernetes deployment.
# If you're using Docker Compose, you don't need this script.
# Instead, use: docker compose up -d
# =================================================================

# Move up one directory to project root
cd "$(dirname "$0")/.." || exit

# Create ConfigMap and Secret
kubectl apply -f k8s/config.yaml

# Create Database StatefulSet and Service
kubectl apply -f k8s/database.yaml
kubectl apply -f k8s/database-service.yaml

# Wait for database to be ready
echo "Waiting for database to be ready..."
kubectl wait --for=condition=ready pod -l app=edgecom-db --timeout=120s

# Create Application Deployment and Service
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

# Check deployment status
echo "Checking deployment status..."
kubectl get pods 