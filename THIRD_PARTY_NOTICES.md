# Avisos de terceiros

## Dependências de execução

Nenhuma dependência externa de execução no momento. O núcleo usa exclusivamente a biblioteca padrão de Go (PRD §3.3).

Cada nova dependência deve ser registrada aqui com nome, versão, licença e motivo, antes de ser adicionada ao `go.mod`. Dependências precisam ser compatíveis com a licença MIT.

## Ferramentas de desenvolvimento

Ferramentas usadas apenas para verificação e não distribuídas com o aplicativo:

| Ferramenta | Versão verificada | Licença | Uso |
|---|---|---|---|
| golangci-lint | 2.15.0 | GPL-3.0 | Lint (`.golangci.yml`) |
| lizard | 1.26.1 | MIT | Complexidade ciclomática |
| semgrep | 1.181.0 | LGPL-2.1 | Análise estática de segurança |
| trivy | 0.76.0 | Apache-2.0 | Vulnerabilidades e segredos no repositório |
| ruff | 0.17.1 | MIT | Lint Python (não aplicável hoje; não há código Python) |

As ferramentas acima são executadas por `scripts/check.sh` e pelo CI. Não entram no binário.
