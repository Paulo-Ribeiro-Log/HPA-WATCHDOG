#!/bin/bash
# Script de exemplo para testar HPA Watchdog

set -e

echo "🧪 HPA Watchdog - Test Script"
echo "=============================="
echo ""

# Cores
RED='\033[0:31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuração
CLUSTER_CONTEXT="${TEST_CLUSTER_CONTEXT:-minikube}"
NAMESPACE="${TEST_NAMESPACE:-default}"
HPA_NAME="${TEST_HPA_NAME:-}"
WITH_PROMETHEUS="${COLLECT_METRICS:-false}"
WITH_HISTORY="${SHOW_HISTORY:-false}"

echo "Configuração:"
echo "  Cluster:    $CLUSTER_CONTEXT"
echo "  Namespace:  $NAMESPACE"
if [ -n "$HPA_NAME" ]; then
  echo "  HPA:        $HPA_NAME"
else
  echo "  HPA:        (todos)"
fi
echo "  Prometheus: $WITH_PROMETHEUS"
echo "  History:    $WITH_HISTORY"
echo ""

# Verifica se binary existe
if [ ! -f "./build/hpa-watchdog" ]; then
    echo -e "${YELLOW}⚠️  Binary não encontrado. Compilando...${NC}"
    make build
    echo ""
fi

# Verifica conectividade kubectl
echo "🔍 Verificando conectividade kubectl..."
if ! kubectl --context="$CLUSTER_CONTEXT" cluster-info &> /dev/null; then
    echo -e "${RED}❌ Falha ao conectar ao cluster: $CLUSTER_CONTEXT${NC}"
    echo ""
    echo "Clusters disponíveis:"
    kubectl config get-contexts
    exit 1
fi
echo -e "${GREEN}✅ Conectado ao cluster${NC}"
echo ""

# Verifica se namespace existe
echo "🔍 Verificando namespace..."
if ! kubectl --context="$CLUSTER_CONTEXT" get namespace "$NAMESPACE" &> /dev/null; then
    echo -e "${RED}❌ Namespace não encontrado: $NAMESPACE${NC}"
    echo ""
    echo "Namespaces disponíveis:"
    kubectl --context="$CLUSTER_CONTEXT" get namespaces
    exit 1
fi
echo -e "${GREEN}✅ Namespace encontrado${NC}"
echo ""

# Lista HPAs disponíveis
echo "📊 HPAs disponíveis no namespace $NAMESPACE:"
kubectl --context="$CLUSTER_CONTEXT" get hpa -n "$NAMESPACE" 2>/dev/null || echo "  (nenhum HPA encontrado)"
echo ""

# Monta comando
CMD="./build/hpa-watchdog test --cluster $CLUSTER_CONTEXT --namespace $NAMESPACE"

if [ -n "$HPA_NAME" ]; then
    CMD="$CMD --hpa $HPA_NAME"
fi

if [ "$WITH_PROMETHEUS" == "true" ]; then
    CMD="$CMD --prometheus"
fi

if [ "$WITH_HISTORY" == "true" ]; then
    CMD="$CMD --history"
fi

CMD="$CMD --verbose"

echo "🚀 Executando teste..."
echo "Comando: $CMD"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Executa
eval "$CMD"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${GREEN}✅ Teste concluído!${NC}"
