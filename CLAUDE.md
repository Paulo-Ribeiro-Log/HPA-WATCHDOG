# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**HPA Watchdog** is an autonomous monitoring system for Kubernetes Horizontal Pod Autoscalers (HPAs) across multiple clusters. It features a rich Terminal UI (TUI) built with Bubble Tea and Lipgloss, providing real-time monitoring, anomaly detection, and centralized alert management.

**Status**: 🟡 Planning Phase
**Target**: Multi-cluster HPA monitoring with Prometheus + Alertmanager integration

## Core Philosophy: KISS (Keep It Simple, Stupid)

**IMPORTANT**: This project follows the KISS principle strictly. When developing:

- **Prefer simplicity over cleverness** - Straightforward code beats "smart" solutions
- **Don't over-engineer** - Build what's needed now, not what might be needed later
- **Avoid premature optimization** - Make it work first, optimize only if proven necessary
- **Use boring technology** - Proven libraries over new/trendy ones
- **Clear over concise** - Readable code trumps shorter code
- **One responsibility per component** - Each module should do one thing well
- **Fail fast and obviously** - Better to crash with clear error than fail silently
- **Configuration over code** - Make behavior configurable instead of hardcoding complex logic

### KISS in Practice

- **Monitoring loop**: Simple goroutine per cluster, no complex scheduling
- **Data storage**: In-memory time-series (5min), optional SQLite for persistence - no complex databases
- **Alert correlation**: Basic grouping by cluster/namespace/HPA - no ML/AI complexity
- **TUI**: Bubble Tea standard patterns - no custom frameworks
- **Error handling**: Clear error messages, graceful degradation - no silent failures

If a solution feels complex, it probably is. Step back and find the simpler approach.

## Architecture

### Three-Layer Data Collection

1. **Kubernetes API** (client-go): HPA configuration, replica counts, deployment info, events
2. **Prometheus API**: Metrics (CPU/Memory/Network) and temporal analysis with PromQL
3. **Alertmanager API**: Existing alert aggregation and silence management

### Hybrid Approach

- **K8s API**: Configuration and state data (min/max replicas, current/desired replicas)
- **Prometheus**: Primary metrics source with native historical data and rich metrics (CPU, Memory, Request Rate, Error Rate, P95 Latency)
- **Alertmanager**: Primary alert source from existing Prometheus rules (70% of alerts)
- **Watchdog Analyzer**: Complementary anomaly detection for patterns not covered by simple PromQL (30% of alerts)

## Core Data Model

### HPASnapshot
Extended snapshot capturing both K8s state and Prometheus metrics:
- K8s data: HPA config, replicas, resource requests/limits, status
- Prometheus data: Current metrics, 5-minute history, extended metrics (request rate, error rate, latency)
- Data source indicator: Prometheus (preferred), Metrics-Server (fallback), or Hybrid

### UnifiedAlert
Combines alerts from both Alertmanager and Watchdog's own detection:
- Source tracking (Alertmanager vs Watchdog)
- Enrichment with HPASnapshot and AlertContext
- Correlation with related alerts
- Silence and acknowledgment support

## Project Structure

```
hpa-watchdog/
├── cmd/
│   └── main.go                    # Entry point
├── internal/
│   ├── monitor/
│   │   ├── collector.go           # Unified collector (K8s + Prometheus + Alertmanager)
│   │   ├── analyzer.go            # Anomaly detection
│   │   └── alerter.go             # Alert system
│   ├── prometheus/
│   │   ├── client.go              # Prometheus API wrapper
│   │   ├── queries.go             # Predefined PromQL queries
│   │   └── discovery.go           # Auto-discovery of endpoints
│   ├── alertmanager/
│   │   └── client.go              # Alertmanager API wrapper
│   ├── storage/
│   │   ├── timeseries.go          # Time-series cache (reduced - Prometheus has history)
│   │   └── persistence.go         # Optional SQLite persistence
│   ├── config/
│   │   ├── loader.go              # Config loading
│   │   ├── thresholds.go          # Threshold management
│   │   └── clusters.go            # Cluster discovery
│   ├── tui/
│   │   ├── app.go                 # Main Bubble Tea app
│   │   ├── views.go               # View rendering
│   │   ├── handlers.go            # Event handlers
│   │   ├── components/            # UI components (dashboard, alerts, charts, config)
│   │   └── styles.go              # Lipgloss styles
│   └── models/
│       └── types.go               # Data structures
├── configs/
│   └── watchdog.yaml              # Default configuration
└── HPA_WATCHDOG_*.md              # Specification documents
```

## Development Commands

### Building
```bash
# Build the binary
go build -o build/hpa-watchdog ./cmd/main.go

# Build with version info
go build -ldflags "-X main.Version=v1.0.0" -o build/hpa-watchdog ./cmd/main.go
```

### Running
```bash
# Run with default config
./build/hpa-watchdog

# Run with custom config
./build/hpa-watchdog --config /path/to/watchdog.yaml

# Debug mode (verbose logging)
./build/hpa-watchdog --debug
```

### Testing
```bash
# Run all tests
go test ./...

# Test specific package
go test ./internal/monitor/...

# Test with coverage
go test -cover ./...

# Integration tests (requires K8s cluster access)
go test ./tests/integration/...
```

### Configuration Validation
```bash
# Validate config file
./build/hpa-watchdog validate --config configs/watchdog.yaml
```

## Key Dependencies

- **k8s.io/client-go@v0.31.4**: Kubernetes API client
- **github.com/charmbracelet/bubbletea@v0.24.2**: TUI framework
- **github.com/charmbracelet/lipgloss@v1.1.0**: Terminal styling
- **github.com/prometheus/client_golang**: Prometheus API client
- **github.com/spf13/viper**: Configuration management
- **github.com/guptarohit/asciigraph**: ASCII charts for metrics
- **github.com/rs/zerolog**: Structured logging
- **github.com/mattn/go-sqlite3** (optional): Persistence

## Important Prometheus Queries

### CPU Usage (HPA Target)
```promql
sum(rate(container_cpu_usage_seconds_total{namespace="{namespace}",pod=~"{pod_selector}"}[1m])) /
sum(kube_pod_container_resource_requests{namespace="{namespace}",pod=~"{pod_selector}",resource="cpu"}) * 100
```

### Replica History
```promql
kube_horizontalpodautoscaler_status_current_replicas{namespace="{namespace}",horizontalpodautoscaler="{name}"}[5m]
```

### Request Rate
```promql
sum(rate(http_requests_total{namespace="{namespace}",service="{service}"}[1m]))
```

### Error Rate
```promql
sum(rate(http_requests_total{namespace="{namespace}",service="{service}",status=~"5.."}[1m])) /
sum(rate(http_requests_total{namespace="{namespace}",service="{service}"}[1m])) * 100
```

## Configuration System

### Config File: `configs/watchdog.yaml`

Key sections:
- **monitoring**: Scan intervals, Prometheus/Alertmanager settings, auto-discovery
- **clusters**: Cluster discovery and filtering
- **storage**: Optional persistence with SQLite
- **alerts**: Source priority, deduplication, correlation
- **thresholds**: CPU/Memory limits, replica deltas, extended metrics
- **ui**: Refresh rate, theme, sounds

### Auto-Discovery

- **Clusters**: Discovers from kubeconfig or `clusters-config.json`
- **Prometheus**: Tries common service patterns in monitoring namespace
- **Alertmanager**: Tries common service patterns in monitoring namespace
- **Fallback**: Uses Kubernetes Metrics-Server if Prometheus unavailable

## Monitoring Loop

Each cluster runs an independent goroutine:
1. List namespaces (skip system namespaces)
2. For each namespace, list HPAs
3. For each HPA:
   - Get config from K8s API
   - Query metrics from Prometheus (current + 5min history)
   - Create HPASnapshot
   - Store in time-series cache
4. Sync alerts from Alertmanager
5. Analyze snapshots for anomalies not covered by Prometheus rules
6. Send unified alerts to TUI via channels
7. Sleep until next scan interval

## Anomaly Detection

### Alertmanager Integration (Primary)
- Syncs existing alerts from Alertmanager API
- Filters HPA-related alerts
- Enriches with context (metrics, history, correlation)
- Provides centralized multi-cluster view
- Allows silence management directly from TUI

### Watchdog Detection (Complementary)
Detects patterns not easily captured by simple PromQL:
- **Replica Oscillation**: Rapid scaling up/down (>5 changes in 5min)
- **Scaling Stuck**: HPA unable to scale when needed
- **Target Deviation**: Current metrics significantly above/below target
- **Config Changes**: HPA min/max or deployment resources modified
- **Complex Correlations**: Multiple metrics indicating systemic issues

## TUI Navigation

### Keyboard Controls
- `Tab`: Switch views (Dashboard, Alerts, Clusters, Config)
- `↑↓` or `j k`: Navigate lists
- `Enter`: View details / Edit
- `A`: Acknowledge alert
- `Shift+A`: Acknowledge all alerts
- `S`: Silence alert (creates Alertmanager silence)
- `C`: Clear acknowledged alerts
- `E`: Enrich alert with metrics context
- `D`: View alert details
- `H`: View snapshot history
- `F5`: Force refresh
- `Ctrl+C` or `Q`: Quit
- `?`: Help

### Views
1. **Dashboard**: Multi-cluster overview, alert summary, ASCII charts, quick stats
2. **Alerts**: Detailed alert list with filtering and correlation
3. **Cluster View**: Per-cluster breakdown by namespace
4. **Config Modal**: Interactive threshold and setting configuration

## Alert Correlation

Watchdog automatically correlates related alerts:
- Groups alerts by cluster/namespace/HPA
- Identifies root cause vs symptoms
- Provides combined analysis across multiple alert types
- Suggests remediation actions

Example: CPU spike → maxed out replicas → high errors → high latency all correlated as single incident.

## Design Principles

1. **Rune-safe**: Always use `[]rune` for Unicode text handling in TUI
2. **Async operations**: Use Bubble Tea commands for async tasks (K8s/Prometheus queries)
3. **Channels for updates**: Monitor goroutines send updates to TUI via channels
4. **Fallback strategy**: Prometheus → Metrics-Server, graceful degradation
5. **Minimal storage**: Leverage Prometheus TSDB instead of heavy local caching
6. **Read-only**: No cluster modifications, safe monitoring operations

## Security & Permissions

### Required K8s RBAC
```yaml
- apiGroups: [""]
  resources: ["namespaces", "pods"]
  verbs: ["get", "list"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets", "statefulsets", "daemonsets"]
  verbs: ["get", "list"]
- apiGroups: ["autoscaling"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["get", "list"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list"]
```

**Note**: All operations are read-only. No write/modify permissions needed.

## Performance Targets

- **Scan Time**: <5s per cluster (50 HPAs, 10 namespaces)
- **Memory Usage**: <100 MB (5 clusters, 250 HPAs, 5min history)
- **CPU Usage**: <5% idle
- **Alertmanager Sync**: 30s interval
- **TUI Refresh**: 500ms

## Roadmap Status

### Phase 1: MVP (Current)
- Project setup and structure
- Core monitoring (K8s + Prometheus)
- Alertmanager client integration
- Basic TUI (dashboard + alerts)
- Config system

### Phase 2: Advanced Features
- Silence management via TUI
- Alert correlation engine
- Enhanced UI with ASCII charts
- Advanced anomaly detection
- SQLite persistence

### Phase 3: Production Ready
- Systemd service file
- Docker image
- Webhook notifications (Slack, Discord, Teams)
- Performance optimization
- Comprehensive testing

## Common Patterns

### Adding a New Anomaly Type
1. Define type in `internal/models/types.go` (`AnomalyType`)
2. Add threshold config in `configs/watchdog.yaml`
3. Implement detection logic in `internal/monitor/analyzer.go`
4. Add TUI rendering in `internal/tui/components/alerts_panel.go`

### Adding a New Prometheus Query
1. Define query template in `internal/prometheus/queries.go`
2. Add parsing logic for result format
3. Integrate into collector in `internal/monitor/collector.go`
4. Update `HPASnapshot` model if new field needed

### Extending TUI Views
1. Create component in `internal/tui/components/`
2. Implement Bubble Tea `Model`, `Update`, and `View` methods
3. Wire into main app in `internal/tui/app.go`
4. Add keyboard handlers in `internal/tui/handlers.go`
5. Define styles in `internal/tui/styles.go`

## Integration with k8s-hpa-manager

While HPA Watchdog can share utility code with the k8s-hpa-manager project (cluster discovery, K8s client wrappers), it is **completely autonomous**:
- Separate binary: `hpa-watchdog`
- Separate config directory: `~/.hpa-watchdog/`
- Independent operation (does not require k8s-hpa-manager running)
- Can run as background daemon or interactive TUI

## Troubleshooting

### Prometheus Connection Issues
- Verify endpoint: `kubectl port-forward -n monitoring svc/prometheus 9090:9090`
- Check auto-discovery patterns in config
- Enable fallback to metrics-server: `prometheus.fallback_to_metrics_server: true`

### Missing Metrics
- Ensure Prometheus is scraping kube-state-metrics
- Check metrics-server is installed: `kubectl top pods`
- Verify HPA target metrics are exposed

### High Memory Usage
- Reduce `history_retention_minutes` (default: 5)
- Limit `max_active_alerts` (default: 100)
- Disable persistence if not needed

### Alertmanager Sync Issues
- Verify Alertmanager endpoint accessibility
- Check alert label filters: `filters.only_hpa_related: true`
- Increase sync interval if rate-limiting occurs
