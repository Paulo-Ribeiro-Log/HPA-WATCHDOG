#!/bin/bash

# HPA Watchdog - Collector Test Script

set -e

echo "🚀 HPA Watchdog - Collector Test"
echo "================================="
echo ""

# Check if binary exists
if [ ! -f "build/test-collector" ]; then
    echo "Building test-collector..."
    go build -o build/test-collector ./cmd/test-collector/
    echo "✅ Build complete"
    echo ""
fi

# Check for cluster context
CLUSTER_CONTEXT="${1:-kind-kind}"
PROMETHEUS_ENDPOINT="${2:-}"

echo "Configuration:"
echo "  Cluster context: $CLUSTER_CONTEXT"
if [ -n "$PROMETHEUS_ENDPOINT" ]; then
    echo "  Prometheus: $PROMETHEUS_ENDPOINT"
else
    echo "  Prometheus: disabled (K8s metrics only)"
fi
echo ""

# Check if context exists
if ! kubectl config get-contexts "$CLUSTER_CONTEXT" &>/dev/null; then
    echo "❌ Error: Cluster context '$CLUSTER_CONTEXT' not found"
    echo ""
    echo "Available contexts:"
    kubectl config get-contexts -o name
    echo ""
    echo "Usage: $0 [cluster-context] [prometheus-endpoint]"
    echo "Example: $0 kind-kind http://localhost:9090"
    exit 1
fi

# Check cluster connectivity
echo "Testing cluster connectivity..."
if ! kubectl --context="$CLUSTER_CONTEXT" cluster-info &>/dev/null; then
    echo "❌ Error: Cannot connect to cluster '$CLUSTER_CONTEXT'"
    exit 1
fi
echo "✅ Cluster is reachable"
echo ""

# Check for HPAs
HPA_COUNT=$(kubectl --context="$CLUSTER_CONTEXT" get hpa --all-namespaces --no-headers 2>/dev/null | wc -l)
echo "HPAs found: $HPA_COUNT"

if [ "$HPA_COUNT" -eq 0 ]; then
    echo ""
    echo "⚠️  Warning: No HPAs found in cluster"
    echo ""
    echo "Would you like to create a test HPA? (y/n)"
    read -r CREATE_TEST_HPA

    if [ "$CREATE_TEST_HPA" = "y" ]; then
        echo ""
        echo "Creating test HPA in namespace 'test'..."

        # Create test namespace
        kubectl --context="$CLUSTER_CONTEXT" create namespace test --dry-run=client -o yaml | kubectl --context="$CLUSTER_CONTEXT" apply -f -

        # Create test deployment
        cat <<EOF | kubectl --context="$CLUSTER_CONTEXT" apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
  namespace: test
spec:
  replicas: 2
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: test-app-hpa
  namespace: test
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: test-app
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
EOF

        echo "✅ Test HPA created"
        echo ""
        echo "Waiting 5 seconds for HPA to initialize..."
        sleep 5
    fi
fi

echo ""
echo "Starting collector..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Run collector
if [ -n "$PROMETHEUS_ENDPOINT" ]; then
    ./build/test-collector "$CLUSTER_CONTEXT" "$PROMETHEUS_ENDPOINT"
else
    ./build/test-collector "$CLUSTER_CONTEXT"
fi
