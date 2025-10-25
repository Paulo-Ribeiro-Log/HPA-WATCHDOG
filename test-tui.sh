#!/bin/bash

# Script para testar a TUI do HPA Watchdog

set -e

echo "🎨 Compilando TUI de teste..."
go build -o build/test-tui ./cmd/test-tui/main.go

echo "🚀 Iniciando TUI..."
echo "   Logs em: /tmp/hpa-watchdog-tui.log"
echo ""
echo "Controles:"
echo "  Tab       - Mudar de view"
echo "  ↑↓ / jk   - Navegar"
echo "  Enter     - Selecionar"
echo "  1-4       - Filtros de severidade"
echo "  Q         - Sair"
echo ""

./build/test-tui
