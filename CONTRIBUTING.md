
## 🎯 Filosofia: KISS

Este projeto segue rigorosamente o princípio **Keep It Simple, Stupid**:

- Prefira simplicidade sobre complexidade
- Código claro é melhor que código "inteligente"
- Não faça over-engineering
- Use tecnologia comprovada

## 🚀 Como Contribuir

### 1. Fork e Clone

```bash
# Fork no GitHub, depois:
git clone https://github.com/SEU_USER/hpa-watchdog.git
cd hpa-watchdog
```

### 2. Crie uma Branch

```bash
git checkout -b feature/minha-feature
# ou
git checkout -b fix/meu-bugfix
```

### 3. Desenvolva

```bash
# Instale dependências
make deps

# Rode testes enquanto desenvolve
make test

# Formate o código
make fmt

# Valide com linter
make lint
```

### 4. Commit

Usamos [Conventional Commits](https://www.conventionalcommits.org/):

```bash
git commit -m "feat: adiciona suporte a custom metrics"
git commit -m "fix: corrige memory leak no collector"
git commit -m "docs: atualiza README com exemplos"
```

Tipos:
- `feat`: Nova feature
- `fix`: Correção de bug
- `docs`: Documentação
- `refactor`: Refatoração
- `test`: Testes
- `chore`: Tarefas de manutenção

### 5. Push e Pull Request

```bash
git push origin feature/minha-feature
```

Depois abra um Pull Request no GitHub com:
- Descrição clara do que foi feito
- Referência a issues relacionadas (se houver)
- Screenshots (se mudança visual)

## 📋 Checklist antes do PR

- [ ] Código formatado (`make fmt`)
- [ ] Linter passou (`make lint`)
- [ ] Testes passando (`make test`)
- [ ] Documentação atualizada (se necessário)
- [ ] Commit messages seguem padrão
- [ ] Branch atualizada com main

## 🧪 Testes

```bash
# Testes unitários
make test

# Testes curtos (sem integração)
make test-short

# Coverage
make coverage
```

Novos recursos devem incluir testes.

## 📝 Código de Conduta

- Seja respeitoso
- Aceite críticas construtivas
- Foque no que é melhor para o projeto
- Mantenha discussões técnicas e objetivas

## 🐛 Reportando Bugs

Abra uma issue com:
- Descrição clara do problema
- Steps para reproduzir
- Comportamento esperado vs atual
- Versão do HPA Watchdog (`./hpa-watchdog version`)
- Ambiente (SO, versão do Go, versão do K8s)

## 💡 Sugerindo Features

Abra uma issue com:
- Descrição clara da feature
- Caso de uso (por que é útil?)
- Proposta de implementação (se tiver)

## 🎨 Style Guide

### Go Code

- Siga [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` (automático no `make fmt`)
- Nomes descritivos (clareza > brevidade)
- Comentários em inglês ou português (consistente)
- Evite abreviações obscuras

### Commits

- Primeira linha: resumo conciso (<50 chars)
- Corpo: detalhes do que e por quê (se necessário)
- Rodapé: referências a issues

### Documentação

- README em português
- Code comments em inglês ou português
- Exemplos práticos sempre que possível

## 🏗️ Estrutura do Projeto

```
internal/
├── monitor/      # Core monitoring logic
├── prometheus/   # Prometheus integration
├── alertmanager/ # Alertmanager integration
├── storage/      # Data storage (time-series, SQLite)
├── config/       # Configuration management
├── tui/          # Terminal UI (Bubble Tea)
└── models/       # Data models
```

## 🆘 Precisa de Ajuda?

- Abra uma issue com sua dúvida
- Marque como `question`
- Seja específico sobre o que precisa

## 📜 Licença

Ao contribuir, você concorda que suas contribuições serão licenciadas sob a MIT License.

---

Obrigado por contribuir! 🚀
