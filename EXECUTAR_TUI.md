# 🖥️ Como Executar a TUI do HPA Watchdog

## ✅ Pré-requisitos

- Terminal interativo (com TTY)
- WSL2 ou Linux
- Go 1.24+ (para compilar)

## 🚀 Opções de Execução

### Opção 1: Windows Terminal (RECOMENDADO para Windows)

1. Abra o **Windows Terminal**
2. Abra uma aba WSL (Ubuntu/Debian)
3. Execute:

```bash
cd /home/paulo/Scripts/Scripts\ GO/HPA-Watchdog
./test-tui.sh
```

### Opção 2: WSL Terminal Direto

No seu terminal WSL favorito (Ubuntu, Debian, etc):

```bash
cd /home/paulo/Scripts/Scripts\ GO/HPA-Watchdog
./build/test-tui
```

### Opção 3: Via tmux (Para sessões persistentes)

```bash
# Instale tmux se necessário
sudo apt install tmux

# Inicie sessão tmux
tmux new -s hpa-watchdog

# Execute a TUI
cd /home/paulo/Scripts/Scripts\ GO/HPA-Watchdog
./build/test-tui

# Para sair preservando: Ctrl+b, depois d
# Para voltar: tmux attach -t hpa-watchdog
```

### Opção 4: SSH com TTY

Se acessar remotamente:

```bash
ssh -t seu-usuario@seu-host "cd /home/paulo/Scripts/Scripts\ GO/HPA-Watchdog && ./build/test-tui"
```

## 🎮 Controles da TUI

| Tecla | Ação |
|-------|------|
| `Tab` | Próxima view |
| `Shift+Tab` | View anterior |
| `↑` `↓` ou `j` `k` | Navegar |
| `Enter` | Selecionar |
| `1` `2` `3` `4` | Filtros (All, Critical, Warning, Info) |
| `R` ou `F5` | Force refresh |
| `Q` ou `Ctrl+C` | Sair |

## 📊 O que você verá

### 1. Dashboard (View inicial)
```
📊 HPA Watchdog - Dashboard                              ● 09:48:32

Dashboard  Alertas  Clusters  Detalhes

┌─────────────────────┐  ┌─────────────────────┐  ┌────────────────────────┐
│ Clusters            │  │ HPAs                │  │ Anomalias              │
│                     │  │                     │  │                        │
│ Total:       3      │  │ Monitorados:   9    │  │ Total:            4    │
│ Online:      3      │  │ Com anomalias: 4    │  │ Critical: 2 Warning: 2 │
└─────────────────────┘  └─────────────────────┘  └────────────────────────┘

📈 Anomalias por Tipo
  CPU_SPIKE      1
  OSCILLATION    1
  ...
```

### 2. Alertas (Tab)
```
🔔 HPA Watchdog - Alertas

Filtros: [1] All  [2] Critical  [3] Warning  [4] Info    Total: 4 anomalias

Hora      Severidade    Tipo              HPA                    Mensagem
────────────────────────────────────────────────────────────────────────
09:48:25  🔴 CRITICAL  HIGH_ERROR_RATE   akspriv.../api-deploy  Taxa de err...
09:48:24  🟡 WARNING   CPU_SPIKE         akspriv.../istio-gw    CPU spike: ...
```

### 3. Clusters (Tab novamente)
```
🏢 HPA Watchdog - Clusters

Status         Cluster                           HPAs  Anomalias  Último Scan
──────────────────────────────────────────────────────────────────────────────
🟢 Online     akspriv-faturamento-prd-admin      3     2          09:48:25
🟢 Online     akspriv-payment-prd-admin          3     1          09:48:26
```

### 4. Detalhes (Enter em uma anomalia)
```
🔍 HPA Watchdog - Detalhes da Anomalia

🔔 CPU_SPIKE  🟡 WARNING

📍 Localização
Cluster:    akspriv-payment-prd-admin
Namespace:  istio-system
HPA:        istio-gateway
Timestamp:  2025-10-25 09:48:24

📝 Descrição
CPU spike: 45.0% → 95.0% (+111.1% em 30s). Aumento abrupto de CPU detectado.

📊 Métricas do HPA
Réplicas atuais:  8
Réplicas min/max: 2 / 10
CPU atual:        95.0%
Memory atual:     70.0%

🔧 Ações Sugeridas
1. Verificar se houve aumento de tráfego súbito
2. Verificar logs da aplicação para erros ou slow queries
3. Monitorar se HPA vai escalar adequadamente
```

## 📝 Dados de Teste

A aplicação de teste gera automaticamente:

- **3 Clusters**: akspriv-faturamento-prd-admin, akspriv-payment-prd-admin, akspriv-api-prd-admin
- **9 HPAs**: nginx-controller, istio-gateway, api-deployment, web-frontend
- **4 Anomalias**: OSCILLATION, CPU_SPIKE, HIGH_ERROR_RATE, MAXED_OUT
- **Atualização**: Novos snapshots a cada 5 segundos

## 🐛 Troubleshooting

### Erro: "could not open a new TTY"

**Causa**: Você está em um ambiente sem terminal interativo (SSH sem -t, CI/CD, Claude Code, etc.)

**Solução**:
```bash
# Se SSH:
ssh -t usuario@host "comando"

# Se local: Abra um terminal real (Windows Terminal, gnome-terminal, etc)
```

### Erro: "permission denied"

```bash
chmod +x build/test-tui
chmod +x test-tui.sh
```

### Cores não aparecem

```bash
export TERM=xterm-256color
./build/test-tui
```

### Terminal muito pequeno

A TUI funciona melhor em terminais de pelo menos 120x40 caracteres.

Redimensione o terminal ou use:
```bash
# Verificar tamanho atual
tput cols; tput lines

# Recomendado: 120 colunas x 40 linhas
```

## 📄 Logs

A aplicação salva logs em:
```
/tmp/hpa-watchdog-tui.log
```

Para ver logs em tempo real:
```bash
# Terminal 2
tail -f /tmp/hpa-watchdog-tui.log
```

## 🎯 Próximos Passos

Depois de testar a TUI:

1. Integrar com o Collector (dados reais dos clusters)
2. Adicionar gráficos ASCII de métricas
3. Implementar exportação de anomalias
4. Adicionar navegação drill-down (Cluster → Namespace → HPA)

## 💡 Dicas

- Use `tmux` para manter a TUI rodando em background
- Maximize o terminal para melhor experiência
- Use Windows Terminal com perfil WSL para melhor renderização
- Ative scroll do terminal para ver mais anomalias

---

**Nota**: Esta é a versão de TESTE com dados simulados. A versão de produção será integrada com o collector real.
