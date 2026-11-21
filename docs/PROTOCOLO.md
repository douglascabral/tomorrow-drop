# Protocolo de transferência — versão 1

Este documento define o protocolo versionado do Tomorrow Drop. A implementação está em `internal/protocol` e os limites em `internal/limits`. Qualquer alteração de formato exige incremento de versão.

## 1. Visão geral

- Transporte: TCP com TLS 1.3 (tarefas 010–011). A autenticação da identidade ocorre na camada de pareamento e confiança, não apenas pelo TLS.
- Descoberta: UDP, datagramas `DISCOVER` e `ANNOUNCE` (tarefa 017, ainda não especificada em detalhes).
- Encoding das mensagens de controle: JSON UTF-8, dentro de quadros com prefixo de tamanho.

## 2. Framing

Cada quadro de controle é:

```
+----------------------+---------------------------+
| tamanho (uint32, BE) | payload JSON (1..64 KiB)  |
+----------------------+---------------------------+
```

- O tamanho é um inteiro sem sinal de 32 bits em big-endian.
- Tamanho zero é inválido e resulta em `ErrMalformed`.
- Tamanho acima de `limits.MaxControlFrameBytes` (64 KiB) resulta em `ErrLimitExceeded`. A rejeição ocorre **antes** da alocação do corpo.
- Corpo truncado ou cabeçalho parcial resultam em `ErrMalformed`.
- Encerramento limpo da conexão, sem nenhum byte de cabeçalho, resulta em `io.EOF`.

Dados de arquivos e textos não passam por esse framing de controle. Seu formato de streaming será definido nas tarefas 028–031.

## 3. Versão e negociação

- Versão atual: `1` (`protocol.CurrentVersion`).
- Cada lado envia a lista de versões que suporta, em ordem qualquer.
- Versão escolhida: a **maior** versão presente nas duas listas.
- Sem versão comum: a conexão é encerrada com `ErrVersionMismatch`.
- Não há downgrade forçado: a versão negociada é sempre uma das versões que o próprio lado local suporta.

Mensagem de abertura (`HELLO`), enviada por ambos os lados após o TLS:

```json
{"type":"HELLO","versions":[1]}
```

## 4. Identificadores de sessão

- `SessionID` tem 16 bytes gerados por `crypto/rand`, exibidos em 32 caracteres hexadecimais.
- Cada conexão tem um único `SessionID`, válido somente para ela.
- O identificador **não** é credencial. A autenticação vem da identidade Ed25519 e do TLS. Confirmações de pareamento devem ser vinculadas ao `SessionID` e às chaves públicas dos dois lados (tarefa 013).

## 5. Estados

Estados de segurança de um par (PRD §10), definidos em `protocol.PeerState`:

| Estado | Rótulo | Significado |
|---|---|---|
| Descoberto | `discovered` | Encontrado por UDP, sem autenticação |
| Conectando | `connecting` | Tentativa de conexão em andamento |
| Aguardando verificação | `awaiting_verification` | TLS provisório; apenas pareamento permitido |
| Confiável | `trusted` | Identidade autenticada e persistida |
| Identidade alterada | `identity_changed` | Chave diferente da armazenada; bloqueado |
| Desconectado | `disconnected` | Sem sessão ativa |

Uma conexão TLS estabelecida **não** leva automaticamente ao estado `trusted`.

## 6. Limites de recursos

Definidos em `internal/limits`:

| Constante | Valor | Uso |
|---|---|---|
| `MaxControlFrameBytes` | 64 KiB | Quadro de controle |
| `MaxDiscoveryDatagram` | 1 KiB | Datagrama UDP de descoberta |
| `MaxFileNameBytes` | 255 | Nome de arquivo |
| `MaxFilesPerOffer` | 1000 | Arquivos por oferta |
| `MaxTextBytes` | 1 MiB | Texto ou link |
| `MaxConcurrentTransfers` | 4 | Transferências simultâneas |
| `MaxDiscoveredDevices` | 256 | Lista efêmera de dispositivos |

## 7. Timeouts

Definidos em `internal/protocol/timeouts.go`:

| Constante | Valor | Uso |
|---|---|---|
| `HandshakeTimeout` | 10 s | TLS e negociação de versão |
| `DecisionTimeout` | 60 s | Aceite ou recusa de recebimento (PRD §13) |
| `IdleTimeout` | 30 s | Conexão sem tráfego |
| `DiscoveryInterval` | 30 s | Consultas periódicas de descoberta (PRD §5.2) |

Valores de `HandshakeTimeout`, `DecisionTimeout` e `IdleTimeout` são decisões de engenharia, não exigências do PRD, e podem ser ajustados com registro nesta seção.

## 8. Erros

Erros de protocolo usam os sentinelas de `internal/apperr`: `ErrMalformed`, `ErrLimitExceeded`, `ErrTimeout`, `ErrCanceled`, `ErrUnauthorized`, `ErrIdentityChanged` e `ErrVersionMismatch`. Mensagens de erro enviadas ao par não incluem dados locais sensíveis.
