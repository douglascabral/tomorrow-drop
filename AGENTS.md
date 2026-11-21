# AGENTS.md — Tomorrow Drop

## Estrutura

- Manter um **monorepo Git com um único módulo Go** (`go.mod` na raiz).
- `cmd/tdrop/`: CLI.
- `cmd/tomorrow-drop/`: aplicativo gráfico Fyne.
- `internal/`: núcleo compartilhado.
- `assets/`: recursos visuais.
- `docs/`: PRD e documentação técnica.
- `scripts/`: automação de build e distribuição.
- Isolar integrações específicas de plataforma com interfaces e *build tags*.
- **Nunca importar Fyne no núcleo compartilhado.**

## Desenvolvimento

- Seguir o PRD e executar `TOMORROW_DROP_TASKS.md` na ordem definida.
- **TDD obrigatório:** escrever o teste primeiro, confirmar que falha, implementar e refatorar.
- Priorizar a biblioteca padrão de Go e minimizar dependências.
- Nunca implementar criptografia própria ou enfraquecer verificações de segurança.
- Manter alterações pequenas, coesas e testáveis.

## Regras inegociáveis de testes

- **Nunca pular, desativar, remover ou enfraquecer um teste para fazer a suíte passar.**
- Proibido usar `skip`, `xfail`, exclusões, mocks artificiais ou alterações de assertions para esconder falhas.
- Se um teste falhar, investigar e corrigir a causa raiz.
- **Todo bug ou erro identificado exige um teste de regressão antes da correção.**
- O teste de regressão deve reproduzir o problema e falhar antes da correção.
- Após corrigir, executar o teste de regressão e a suíte completa.
- Manter permanentemente os testes de regressão para impedir que o bug retorne.
- Nunca declarar sucesso com testes falhando ou verificações não executadas.

## Validação obrigatória

Ao concluir **cada bloco de tarefas**, executar:

```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
ruff check .
lizard .
semgrep scan --config auto .
trivy fs .
```

- Corrigir problemas encontrados antes de avançar.
- Ruff é aplicável apenas quando houver código Python; caso contrário, registrar como não aplicável.
- Investigar e documentar falsos positivos, sem desabilitar verificações indiscriminadamente.
- Não marcar tarefas como concluídas enquanto houver falhas relevantes.
- Atualizar `[   ]` para `[x]` somente após implementação e validação.

## Critério de conclusão

Um bloco estará concluído somente quando:

- A implementação atender ao PRD.
- Os testes TDD e de regressão passarem.
- A suíte completa estiver verde.
- As ferramentas de qualidade e segurança tiverem sido executadas e seus resultados tratados.
- A documentação necessária estiver atualizada.

**Regra final: corrigir o código, nunca maquiar os testes.**