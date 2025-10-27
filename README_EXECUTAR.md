# 🚀 Como Executar o HPA Watchdog

## ⚠️ IMPORTANTE: Diferença entre Aplicações

Este projeto possui **DUAS aplicações**:

### 1. **hpa-watchdog** (APLICAÇÃO REAL) ✅
- **Binário**: `build/hpa-watchdog`
- **Código**: `cmd/hpa-watchdog/main.go`
- **Uso**: Produção e testes reais
- **Características**:
  - Respeita configuração do setup wizard
  - Escaneia APENAS clusters selecionados pelo usuário
  - Carrega clusters do `~/.kube/config`
  - Não gera dados fake

### 2. **test-tui** (APENAS PARA TESTES DE UI) ❌
- **Binário**: `build/test-tui`
- **Código**: `cmd/test-tui/main.go`
- **Uso**: Testar interface sem ter clusters configurados
- **Características**:
  - Gera dados MOCK (fake)
  - Clusters: akspriv-api-prd-admin, akspriv-payment-prd-admin, akspriv-faturamento-prd-admin
  - **IGNORA** configuração do setup
  - **IGNORA** clusters do kubeconfig
  - Apenas para testar navegação e UI

---

## 📋 Executar Aplicação REAL

### Método 1: Makefile (Recomendado)

```bash
cd /home/paulo/Scripts/Scripts\ GO/HPA-Watchdog

# Compilar
make build

# Executar
make run
```

### Método 2: Direto pelo Binário

```bash
cd /home/paulo/Scripts/Scripts\ GO/HPA-Watchdog

# Compilar
make build

# Executar
./build/hpa-watchdog
```

### Método 3: go run

```bash
cd /home/paulo/Scripts/Scripts\ GO/HPA-Watchdog
go run ./cmd/hpa-watchdog/main.go
```

---

## 🎮 Fluxo de Uso

1. **Execute** `./build/hpa-watchdog`
2. **Setup Wizard** será exibido:
   - Escolha modo (Full/Individual/StressTest)
   - Configure clusters, intervalo, duração
   - Confirme
3. **Scan inicia** apenas após confirmação
4. **Dashboard** mostra métricas dos clusters SELECIONADOS
5. **Tecla P** pausa/retoma scan
6. **Ctrl+C ou Q** sai da aplicação

---

## 🧪 Testar Apenas a Interface (MOCK)

Se você quer apenas testar a interface SEM configurar clusters:

```bash
# Compilar versão de teste
make build-test-tui

# Executar (gera dados fake)
./build/test-tui
```

**Nota**: Esta versão NÃO respeita a configuração e gera dados aleatórios apenas para testes de UI.

---

## 🔍 Verificar qual Aplicação Está Rodando

Se você ver dados de clusters que **NÃO selecionou**, está rodando o `test-tui` ao invés do `hpa-watchdog`.

**Clusters MOCK do test-tui:**
- akspriv-api-prd-admin ❌
- akspriv-payment-prd-admin ❌
- akspriv-faturamento-prd-admin ❌

**Clusters REAIS do hpa-watchdog:**
- Apenas os que você tem no `~/.kube/config` ✅
- Apenas os que você selecionou no setup ✅

---

## 📦 Arquivos

- `build/hpa-watchdog` → **Aplicação REAL (use esta)**
- `build/test-tui` → Teste de UI com dados MOCK
- `~/.kube/config` → Fonte dos clusters reais

---

## 🐛 Troubleshooting

### "Clusters que não selecionei aparecem"
→ Você está rodando `./build/test-tui`. Execute `./build/hpa-watchdog`

### "Não carrega meus clusters do kubeconfig"
→ Verifique que `~/.kube/config` existe e tem contextos

### "Setup não funciona"
→ Apenas `hpa-watchdog` usa o setup. O `test-tui` ignora

---

## ✅ Resumo

| Característica | hpa-watchdog | test-tui |
|----------------|--------------|----------|
| Usa setup wizard | ✅ Sim | ❌ Não |
| Carrega kubeconfig | ✅ Sim | ❌ Não |
| Respeita seleção | ✅ Sim | ❌ Não |
| Dados reais | ✅ Sim | ❌ Fake |
| Para produção | ✅ Sim | ❌ Não |
| Para testar UI | ❌ Não | ✅ Sim |

**SEMPRE use `./build/hpa-watchdog` para trabalho real!**
