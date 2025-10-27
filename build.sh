#!/bin/bash

# Build script for HPA Watchdog
# Compila todos os binários na pasta ./build/

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Informações do build
VERSION=${VERSION:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Diretórios
BUILD_DIR="./build"
CMD_DIR="./cmd"

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}  HPA Watchdog - Build Script  ${NC}"
echo -e "${BLUE}================================${NC}"
echo ""
echo -e "${YELLOW}Version:${NC}    $VERSION"
echo -e "${YELLOW}Commit:${NC}     $COMMIT"
echo -e "${YELLOW}Build Date:${NC} $BUILD_DATE"
echo ""

# Cria diretório de build se não existir
if [ ! -d "$BUILD_DIR" ]; then
    echo -e "${YELLOW}Criando diretório $BUILD_DIR...${NC}"
    mkdir -p "$BUILD_DIR"
fi

# Limpa builds anteriores
echo -e "${YELLOW}Limpando builds anteriores...${NC}"
rm -f "$BUILD_DIR"/*

# Build flags
LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X main.Version=$VERSION"
LDFLAGS="$LDFLAGS -X main.Commit=$COMMIT"
LDFLAGS="$LDFLAGS -X main.BuildDate=$BUILD_DATE"

# Função para compilar um binário
build_binary() {
    local name=$1
    local path=$2
    
    echo -e "${BLUE}Compilando $name...${NC}"
    
    if go build -ldflags "$LDFLAGS" -o "$BUILD_DIR/$name" "$path"; then
        # Obtém tamanho do binário
        size=$(du -h "$BUILD_DIR/$name" | cut -f1)
        echo -e "${GREEN}✓ $name compilado com sucesso${NC} (${size})"
    else
        echo -e "${RED}✗ Erro ao compilar $name${NC}"
        return 1
    fi
}

echo ""
echo -e "${YELLOW}Iniciando compilação...${NC}"
echo ""

# Compila aplicação principal
build_binary "hpa-watchdog" "./cmd/hpa-watchdog"

# Compila utilitários de teste
build_binary "test-collector" "./cmd/test-collector"
build_binary "test-tui" "./cmd/test-tui"

echo ""
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}  Build concluído com sucesso! ${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo -e "${YELLOW}Binários gerados em:${NC} $BUILD_DIR/"
echo ""
ls -lh "$BUILD_DIR"
echo ""
echo -e "${BLUE}Para executar:${NC}"
echo -e "  ${YELLOW}./build/hpa-watchdog${NC}        - Aplicação principal"
echo -e "  ${YELLOW}./build/test-collector${NC}     - Teste do coletor"
echo -e "  ${YELLOW}./build/test-tui${NC}           - Teste da interface TUI"
echo ""
