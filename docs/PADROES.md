# Padrões do núcleo

Padrões comuns a todos os pacotes de `internal/`. Estes pontos foram definidos na tarefa 004.

## Erros

- Erros sentinela ficam em `internal/apperr`.
- Envolva com contexto usando `fmt.Errorf("operação: %w", err)`.
- Verifique com `errors.Is`, nunca comparando strings.
- Mensagens não incluem conteúdo transferido, chaves, textos da área de transferência ou caminhos completos.
- Erros de `context` devem passar por `apperr.FromContext` antes de sair de uma operação.

## Logging

- Use `log/slog` pelo construtor `safelog.New`, que redige automaticamente os campos sensíveis.
- Campos sensíveis são identificados pelo nome: `private`, `secret`, `token`, `password`, `key`, `content`, `clipboard`, `payload`, `filename` e `path`. Valores `[]byte` também são redigidos.
- Nomes de campos não sensíveis, como `device`, `fingerprint` e `context`, são preservados.
- Nunca registre a chave privada, mesmo truncada.

## Cancelamento

- Toda operação bloqueante recebe `context.Context` como primeiro parâmetro.
- Operações de rede e de I/O longo verificam o contexto a cada passo.
- Fechar o aplicativo cancela o contexto raiz (PRD §17.3).

## Limites de recursos

- Todos os limites ficam em `internal/limits`. Não existem números mágicos de tamanho ou quantidade espalhados pelo código.
- Tamanhos vindos da rede são validados **antes** de qualquer alocação.

## Testes

- Escreva o teste primeiro e confirme que ele falha (TDD, AGENTS.md).
- Bugs exigem teste de regressão que reproduza o problema antes da correção, mantido permanentemente.
- Não use `skip`, mocks artificiais nem alterações de asserções para esconder falhas.
