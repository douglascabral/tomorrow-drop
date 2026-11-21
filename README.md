# Tomorrow Drop

**Transferiu. Usou. Pronto.**

Aplicativo livre e minimalista para transferir arquivos, textos, links e conteúdo da área de transferência entre dispositivos na mesma rede local. Não usa contas, nuvem, servidores centrais nem conexão com a internet.

> **Status:** fundação do projeto (bloco 01). Descoberta, pareamento, transferência e interfaces ainda não estão implementados. Ver [TOMORROW_DROP_TASKS.md](TOMORROW_DROP_TASKS.md).

## Documentação

- [PRD v1.9](docs/PRD.md): requisitos, decisões de produto e critérios de aceitação.
- [Compilação](docs/COMPILACAO.md): build para Linux, Windows e macOS.
- [Protocolo](docs/PROTOCOLO.md): framing, versão, estados, limites e timeouts.
- [Padrões do núcleo](docs/PADROES.md): erros, logging, cancelamento e testes.
- [Qualidade](docs/QUALIDADE.md): verificações e ferramentas.
- [Avisos de terceiros](THIRD_PARTY_NOTICES.md): dependências e licenças.

## Estrutura

```
cmd/tdrop/            CLI
cmd/tomorrow-drop/    aplicativo gráfico (Fyne, a integrar na tarefa 045)
internal/             núcleo compartilhado entre GUI e CLI
assets/               recursos visuais
docs/                 PRD e documentação técnica
scripts/              automação de verificação e distribuição
```

Pacotes do núcleo: `identity`, `trust`, `discovery`, `transport`, `protocol`, `transfer`, `storage`, `lifecycle`, `ui`, além de `apperr`, `safelog` e `limits` (base comum).

Regra de arquitetura: o núcleo nunca importa Fyne. Integrações específicas de plataforma ficam atrás de interfaces e *build tags*.

## Desenvolvimento

Requisitos: Go 1.24 ou superior.

```sh
go test ./...
go test -race ./...
```

Para executar todas as verificações de qualidade e segurança, veja [docs/QUALIDADE.md](docs/QUALIDADE.md).

## Licença

[MIT](LICENSE).
