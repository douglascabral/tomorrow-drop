
# Tomorrow Drop — Tasks de implementação

> Ordem de execução obrigatória: concluir e validar uma task antes de iniciar a próxima, salvo quando a task declarar explicitamente independência. Marcar `[x]` somente após implementação, testes e documentação correspondentes. Não alterar decisões do PRD v1.9 sem aprovação. Não inventar garantias de segurança nem substituir verificação de identidade por TOFU.

## 01 — Fundação do projeto

[x] 001. Criar repositório Go com `go.mod`, estrutura modular (`cmd/tdrop`, `cmd/tomorrow-drop`, `internal/identity`, `internal/trust`, `internal/discovery`, `internal/transport`, `internal/protocol`, `internal/transfer`, `internal/storage`, `internal/lifecycle`, `internal/ui`), README e `.gitignore`.
[x] 002. Adicionar licença MIT, avisos de dependências e documentação de compilação inicial para Linux, Windows e macOS.
[   ] 003. Configurar lint, formatação (`gofmt`), testes (`go test ./...`), análise estática (`go vet ./...`) e pipeline CI multiplataforma.
[x] 004. Definir padrões de erros, logging sem dados sensíveis, cancelamento com `context.Context`, limites de recursos e testes de unidade.
[x] 005. Definir e documentar protocolo versionado: formatos de mensagens, framing, limites de tamanho, timeouts, identificadores de sessão, estados e negociação de versão.

## 02 — Identidade e confiança

[   ] 006. Implementar geração de identidade Ed25519 com `crypto/rand`, armazenamento persistente e permissões restritivas; não registrar a chave privada em logs.
[   ] 007. Implementar carregamento e validação da identidade existente; tratar corrupção, ausência e regeneração deliberada sem sobrescrever silenciosamente.
[   ] 008. Implementar fingerprint SHA-256 dos 32 bytes canônicos da chave pública Ed25519, exibindo todos os 64 caracteres hexadecimais em grupos legíveis.
[   ] 009. Implementar armazenamento local de chaves públicas confiáveis, nomes amigáveis e revogação manual, com gravação segura e testes de persistência.
[   ] 010. Implementar emissão e validação de certificados TLS vinculados à identidade Ed25519, usando APIs nativas de Go.
[   ] 011. Implementar TLS 1.3 com autenticação criptográfica das identidades dos dois lados, validação de certificados e pinning; proibir aceitação insegura silenciosa em conexões confiáveis.
[   ] 012. Implementar estado TLS provisório para primeiro contato: criptografado, porém sem confiança, com superfície de mensagens estritamente limitada ao pareamento.
[   ] 013. Implementar fluxo de pareamento mútuo com comparação cruzada dos fingerprints completos e confirmações explícitas nos dois dispositivos, vinculadas à mesma sessão e às mesmas chaves.
[   ] 014. Impedir transferências e comandos privilegiados antes da conclusão do pareamento; persistir confiança somente após confirmação mútua comprovada.
[   ] 015. Rejeitar mudanças inesperadas de identidade e impedir downgrade, reutilização de confirmação ou confusão entre sessões.
[   ] 016. Criar testes de segurança para MITM, substituição de chaves/certificados, sessão provisória, confirmação unilateral, revogação e reconexão.

## 03 — Descoberta e conectividade

[   ] 017. Definir formato UDP versionado para `DISCOVER` e `ANNOUNCE`, com validação estrita, tamanho máximo e ausência de informações sensíveis.
[   ] 018. Implementar listener de descoberta UDP enquanto o aplicativo estiver disponível, com respostas preferencialmente unicast, limitação por origem e atraso aleatório curto.
[   ] 019. Implementar envio de `DISCOVER` imediatamente ao abrir/ativar o aplicativo e ao clicar em Atualizar.
[   ] 020. Implementar consultas periódicas a cada 30 segundos somente enquanto o aplicativo estiver visível e disponível; reiniciar o contador após Atualizar e limitar cliques repetidos.
[   ] 021. Suspender consultas periódicas durante transferências e interromper descoberta quando o aplicativo estiver minimizado ou em segundo plano.
[   ] 022. Implementar lista efêmera de dispositivos encontrados, atualização/expiração de entradas e distinção visual entre descoberto e confiável.
[   ] 023. Implementar conexão manual por IP:porta com validação de entrada e a mesma autenticação TLS do fluxo automático.
[   ] 024. Testar descoberta e conectividade em Wi-Fi doméstico e redes empresariais: mesma VLAN, broadcast bloqueado, isolamento de clientes, múltiplas interfaces e ausência de rota.

## 04 — Protocolo e transferência

[   ] 025. Implementar máquina de estados de conexão e transferência, com cancelamento, timeouts, limites de concorrência e limpeza determinística.
[   ] 026. Definir metadados de oferta (tipo, nomes, tamanhos e quantidade), validação de caminhos e limites para impedir traversal e exaustão de recursos.
[   ] 027. Implementar solicitação de recebimento com identidade autenticada e aceite/recusa obrigatórios para **cada** transferência, inclusive entre dispositivos confiáveis.
[   ] 028. Implementar transferência de texto e links via TLS 1.3 autenticado, sem histórico persistente.
[   ] 029. Implementar envio de arquivo único em streaming, sem carregar todo o conteúdo na memória.
[   ] 030. Implementar envio de múltiplos arquivos, com progresso individual e total e comportamento definido para falha parcial.
[   ] 031. Implementar gravação de recebimentos em arquivos temporários com nomes imprevisíveis, validação de tamanho, hash de conteúdo e confirmação de conclusão.
[   ] 032. Implementar publicação segura do arquivo concluído, evitando sobrescrita silenciosa e colisões de nomes; usar operação atômica quando suportada.
[   ] 033. Implementar cancelamento, erro, timeout e queda de conexão: invalidar transferência, excluir parciais e exigir reenvio desde o primeiro byte.
[   ] 034. Implementar limpeza de arquivos temporários abandonados por encerramentos inesperados, sem apagar arquivos alheios ao aplicativo.
[   ] 035. Implementar salvamento padrão em Downloads no desktop e abstração de armazenamento para mecanismos nativos de Android e iOS.
[   ] 036. Criar testes de integridade, arquivos grandes, arquivos vazios, múltiplos arquivos, falhas aos 99%, nomes maliciosos, limites e concorrência.

## 05 — CLI

[   ] 037. Implementar `tdrop identity show`, exibindo identidade local e fingerprint completo sem revelar chave privada.
[   ] 038. Implementar `tdrop discover` e conexão manual por IP:porta.
[   ] 039. Implementar `tdrop pair <destino>` com confirmação interativa e verificação mútua.
[   ] 040. Implementar `tdrop trust list` e `tdrop trust remove <id>`.
[   ] 041. Implementar `tdrop send <arquivo> --to <destino>` e `tdrop send --stdin --to <destino>`.
[   ] 042. Implementar `tdrop receive` com autorização interativa obrigatória por padrão.
[   ] 043. Implementar `tdrop daemon` como modo contínuo explicitamente iniciado, documentando regras de autorização não interativa quando configuradas.
[   ] 044. Criar testes de integração CLI↔CLI, códigos de saída, stdout/stderr, cancelamento e execução headless.

## 06 — Interface desktop Fyne

[   ] 045. Criar aplicativo Fyne com identidade visual minimalista: logotipo de duas montanhas e sol, tema escuro, hierarquia simples e componentes acessíveis.
[   ] 046. Implementar tela inicial com Enviar, Receber, dispositivos encontrados, Atualizar, conexão manual e configurações.
[   ] 047. Implementar tela de pareamento com fingerprints completos de ambos os dispositivos, orientação de conferência cruzada, confirmação explícita e cancelamento.
[   ] 048. Implementar tela de dispositivos confiáveis, inspeção de fingerprint e remoção de confiança.
[   ] 049. Implementar seleção de arquivos, composição de texto e prévia do conteúdo antes do envio.
[   ] 050. Implementar ação manual “Enviar área de transferência” para texto e links; adicionar imagens copiadas somente onde as APIs suportarem, sem monitoramento automático.
[   ] 051. Implementar diálogo de recebimento com identidade do remetente, nomes, tamanhos e botões Aceitar/Recusar.
[   ] 052. Implementar progresso, cancelamento, erros claros e tela de conclusão, sem histórico persistente.
[   ] 053. Implementar comportamento desktop: visível permite descoberta/listener; minimizado fecha listener e descoberta, mas mantém somente transferências previamente autorizadas até terminarem/falharem/cancelarem; fechar o app cancela tudo.
[   ] 054. Testar GUI desktop em Linux, Windows e macOS, incluindo minimizar durante transferência, retorno à janela, encerramento e permissões de arquivos.

## 07 — Android e iOS/iPadOS

[   ] 055. Preparar builds Fyne para Android e iOS/iPadOS, identificando requisitos de toolchain, assinatura e permissões mínimas de rede local.
[   ] 056. Integrar descoberta UDP e TLS aos eventos de primeiro plano/segundo plano em Android e iOS; não criar serviço de transferência em background.
[   ] 057. Implementar interrupção de descoberta, pareamento e transferências ao entrar em segundo plano ou bloquear a tela, com limpeza de recursos e recomeço do zero.
[   ] 058. Integrar seleção e salvamento de arquivos por mecanismos nativos: Downloads quando viável no Android e localização acessível pelo app Arquivos/exportação no iOS.
[   ] 059. Integrar leitura manual de clipboard no mobile, sem monitoramento e sem permissão de câmera; validar suporte real a texto, links e imagens.
[   ] 060. Adaptar telas Fyne para toque, diferentes tamanhos de tela, acessibilidade e orientação.
[   ] 061. Executar testes em dispositivos reais Android↔desktop, iOS↔desktop e Android↔iOS, incluindo permissões, suspensão e rede corporativa.

## 08 — Segurança, qualidade e lançamento

[   ] 062. Revisar modelo de ameaças, estados de autenticação, validação TLS e pinning; executar testes adversariais do primeiro pareamento.
[   ] 063. Aplicar fuzzing aos parsers UDP, mensagens de protocolo, metadados e framing; verificar limites de memória e CPU.
[   ] 064. Revisar armazenamento de chaves privadas, permissões, logs, arquivos temporários, tratamento de erros e proteção contra sobrescrita.
[   ] 065. Validar interoperabilidade completa GUI↔GUI, GUI↔CLI e CLI↔CLI em plataformas suportadas.
[   ] 066. Executar testes de desempenho e estabilidade para arquivos grandes, conexões lentas, falhas de rede e muitos dispositivos em uma VLAN.
[   ] 067. Revisar dependências, licenças e cadeia de compilação; gerar lista de dependências e instruções reproduzíveis de build.
[   ] 068. Documentar instalação, pareamento seguro, conexão manual, comportamento de 30 segundos, clipboard, Downloads, ciclo de vida e limites do aplicativo.
[   ] 069. Preparar empacotamento e distribuição para Linux, Windows, macOS, Android e iOS/iPadOS, incluindo assinatura quando exigida.
[   ] 070. Executar checklist final do PRD v1.9 e registrar evidências de aprovação de todos os critérios de aceitação do MVP.
