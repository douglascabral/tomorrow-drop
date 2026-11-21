# Tomorrow Drop
## Product Requirements Document (PRD)

**Versão:** 1.9  
**Status:** Aprovado para implementação  
**Slogan:** *Transferiu. Usou. Pronto.*  
**Licença:** MIT  
**Linguagem:** Go  
**Interface gráfica:** Fyne  
**Plataformas:** Linux, Windows, macOS, Android e iOS/iPadOS

---

# 1. Visão geral

Tomorrow Drop é um aplicativo livre, minimalista e multiplataforma para transferência direta de arquivos, textos, links, imagens e conteúdo da área de transferência entre dispositivos conectados à mesma rede local.

O aplicativo não depende de contas, serviços em nuvem, servidores centrais ou conexão com a internet.

O fluxo principal é simples:

1. Abrir o Tomorrow Drop.
2. Encontrar outro dispositivo na rede local.
3. Realizar o pareamento seguro, caso ainda não exista confiança.
4. Escolher o conteúdo.
5. Enviar.
6. O destinatário aceita ou recusa.
7. A transferência é concluída.

**Princípio do produto:** Transferiu. Usou. Pronto.

# 2. Objetivos

O Tomorrow Drop deverá:

- Permitir transferências diretas entre computadores e dispositivos móveis.
- Funcionar sem infraestrutura externa.
- Utilizar criptografia e autenticação de identidade.
- Manter uma interface visual simples e consistente.
- Minimizar permissões do sistema operacional.
- Reduzir dependências externas.
- Evitar processos residentes desnecessários.
- Operar em redes domésticas e empresariais.
- Oferecer uma CLI para automação e ambientes sem interface gráfica.
- Respeitar a privacidade do usuário.

## 2.1 Fora do escopo

Não serão implementados no MVP:

- Contas de usuário.
- Sincronização em nuvem.
- Servidores de relay.
- Transferências pela internet.
- Histórico persistente de transferências.
- Sincronização automática do clipboard.
- Retomada de transferências interrompidas.
- QR Code.
- Pareamento por câmera.
- Serviços de transferência em segundo plano no mobile.
- Descoberta automática entre VLANs diferentes.
- Atualização automática de chaves de identidade.

# 3. Plataformas e tecnologias

## 3.1 Sistemas operacionais

O aplicativo terá suporte a:

- Linux.
- Windows.
- macOS.
- Android.
- iOS/iPadOS.

## 3.2 Linguagem e interface

**Go** será a linguagem principal.

**Fyne** será utilizado para a interface gráfica multiplataforma.

A interface de linha de comando, denominada `tdrop`, também será implementada em Go.

O núcleo do aplicativo deverá ser compartilhado entre a GUI e a CLI.

## 3.3 Bibliotecas nativas

Sempre que possível, utilizar a biblioteca padrão de Go:

- `crypto/tls`
- `crypto/ed25519`
- `crypto/sha256`
- `crypto/rand`
- `crypto/x509`
- `net`
- `context`
- `io`
- `os`

Dependências externas serão utilizadas apenas quando necessárias e deverão ser avaliadas quanto a manutenção, segurança e compatibilidade com a licença MIT.

# 4. Arquitetura

O Tomorrow Drop terá uma arquitetura modular.

## 4.1 Núcleo compartilhado

O núcleo será responsável por:

1. Identidade criptográfica.
2. Descoberta UDP.
3. Conexões TCP/TLS.
4. Pareamento.
5. Armazenamento de confiança.
6. Protocolo de transferência.
7. Gerenciamento de sessões.
8. Controle do ciclo de vida.
9. Tratamento de erros.
10. Limpeza de recursos.

## 4.2 Interfaces

A GUI Fyne e a CLI utilizarão o mesmo núcleo.

As regras de autenticação, transferência e segurança não poderão divergir entre as interfaces.

## 4.3 Comunicação

A comunicação será direta entre dispositivos.

Não haverá servidor central, conta, nuvem, relay ou dependência obrigatória de serviços externos.

# 5. Descoberta de dispositivos

## 5.1 Protocolo

A descoberta utilizará UDP broadcast na rede local.

O protocolo terá dois tipos lógicos de mensagens:

**DISCOVER**

Solicita informações sobre dispositivos disponíveis.

**ANNOUNCE**

Responde à solicitação de descoberta, preferencialmente por UDP unicast ao solicitante.

Não haverá necessidade de anúncios periódicos de presença enviados por todos os dispositivos.

## 5.2 Intervalo de descoberta

**Intervalo aprovado: 30 segundos.**

| Evento | Comportamento |
|---|---|
| Aplicativo aberto | Enviar DISCOVER imediatamente |
| Aplicativo visível e disponível | Enviar DISCOVER a cada 30 segundos |
| Usuário clica em Atualizar | Enviar DISCOVER imediatamente |
| Retorno ao primeiro plano | Enviar DISCOVER imediatamente |
| Desktop minimizado | Suspender descoberta |
| Mobile em segundo plano | Suspender descoberta |
| Transferência em andamento | Suspender consultas periódicas |
| Aplicativo encerrado | Nenhuma atividade de descoberta |

Ao clicar em Atualizar, o contador de 30 segundos será reiniciado.

Cliques repetidos serão limitados para evitar consultas excessivas.

## 5.3 Respostas

Os dispositivos disponíveis deverão responder às consultas recebidas, respeitando:

- Limitação de frequência.
- Pequeno atraso aleatório para distribuir respostas.
- Tamanho máximo de mensagens.
- Validação do formato.
- Rejeição de mensagens malformadas.

A implementação deverá evitar amplificação desnecessária de tráfego UDP.

## 5.4 Redes empresariais

A descoberta deverá respeitar o domínio local de broadcast.

O aplicativo não tentará atravessar VLANs, contornar firewalls ou burlar isolamento de clientes.

Redes empresariais podem impedir a descoberta ou a comunicação direta entre dispositivos.

Nessas situações, será disponibilizada a conexão manual.

## 5.5 Conexão manual

O usuário poderá informar:

- Endereço IP.
- Porta TCP.

A conexão manual estará sujeita às mesmas verificações criptográficas da descoberta automática.

O endereço IP não será utilizado como identidade confiável.

## 5.6 Segurança da descoberta

Anúncios UDP serão considerados não confiáveis.

Nenhuma informação de descoberta será suficiente para autorizar uma transferência.

O dispositivo deverá ser autenticado pela identidade criptográfica antes de receber confiança.

# 6. Identidade criptográfica

Cada instalação terá um par de chaves Ed25519 persistente.

A chave pública identificará criptograficamente o dispositivo.

A chave privada deverá permanecer local e ser protegida pelos mecanismos adequados de cada plataforma.

A identidade não dependerá de:

- Nome do dispositivo.
- Endereço IP.
- Endereço MAC.
- Interface de rede.
- Nome de usuário do sistema.

Uma reinstalação que gere uma nova chave será tratada como uma nova identidade.

# 7. Fingerprint

O fingerprint será calculado sobre a representação canônica de 32 bytes da chave pública Ed25519.

**Algoritmo:**

`SHA-256(chave pública Ed25519 canônica)`

O resultado terá 32 bytes, apresentados como 64 caracteres hexadecimais.

A interface deverá:

- Exibir o fingerprint completo.
- Agrupar os caracteres para facilitar a leitura.
- Identificar claramente a qual dispositivo pertence cada fingerprint.
- Evitar truncamento.
- Permitir conferência visual.

Não serão utilizados códigos curtos como substitutos da verificação completa.

# 8. Transporte seguro

As conexões de transferência utilizarão **TLS 1.3**.

A implementação deverá garantir:

- Criptografia do tráfego.
- Integridade das mensagens.
- Autenticação da identidade criptográfica.
- Verificação de posse da chave privada.
- Associação entre a identidade Ed25519 e a credencial apresentada no TLS.
- Verificação da chave pública previamente confiável em reconexões.

## 8.1 Conexão provisória

No primeiro contato, uma conexão TLS poderá ser estabelecida antes da autenticação fora de banda.

Essa conexão será considerada provisória.

Embora criptografada, ela não será considerada autenticada até a confirmação do pareamento.

Nenhuma transferência de conteúdo será autorizada nesse estado.

## 8.2 Reconexões

Dispositivos já pareados deverão ser autenticados mediante comparação com a chave pública persistida.

Chaves diferentes das esperadas resultarão em bloqueio.

Não haverá aceitação automática de alterações de identidade.

# 9. Pareamento inicial

O pareamento será realizado exclusivamente por **comparação manual de fingerprints completos**.

Não haverá QR Code, câmera, PAKE ou códigos SAS curtos.

## 9.1 Fluxo

1. Dois dispositivos estabelecem uma conexão TLS provisória.
2. Cada dispositivo identifica a chave pública apresentada pelo outro.
3. Cada dispositivo calcula seu fingerprint local.
4. Cada dispositivo calcula o fingerprint da identidade remota.
5. As duas telas exibem as identidades local e remota.
6. O usuário compara os fingerprints de forma cruzada.
7. Cada dispositivo exige confirmação explícita.
8. As confirmações são vinculadas à mesma sessão e às mesmas identidades.
9. Após a confirmação mútua, as chaves públicas são armazenadas como confiáveis.
10. A conexão passa a permitir operações autorizadas.

## 9.2 Comparação cruzada

O fingerprint local do dispositivo A deverá corresponder ao fingerprint remoto exibido no dispositivo B.

O fingerprint local do dispositivo B deverá corresponder ao fingerprint remoto exibido no dispositivo A.

Ambas as comparações são obrigatórias.

## 9.3 Confirmação

A interface deverá apresentar instruções claras.

A confirmação não poderá ser interpretada como automática ou implícita.

O protocolo deverá impedir que confirmações de sessões diferentes sejam reutilizadas.

## 9.4 Persistência

**Decisão aprovada:** a confiança permanecerá válida até remoção manual.

Não haverá expiração automática por tempo ou inatividade.

## 9.5 Revogação

O usuário poderá remover um dispositivo confiável.

Após a revogação, uma nova conexão exigirá novo pareamento.

# 10. Estados de segurança

O aplicativo deverá distinguir:

**Descoberto:** dispositivo encontrado, sem autenticação.

**Conectando:** tentativa de estabelecer conexão.

**Aguardando verificação:** conexão provisória, aguardando pareamento.

**Confiável:** identidade autenticada e persistida.

**Identidade alterada:** chave pública diferente da armazenada; conexão bloqueada.

**Desconectado:** sem sessão ativa.

O aplicativo não deverá apresentar uma conexão como confiável apenas porque o TLS foi estabelecido.

# 11. Envio de conteúdo

O usuário poderá enviar:

- Arquivos individuais.
- Múltiplos arquivos.
- Textos.
- Links.
- Conteúdo da área de transferência.
- Imagens copiadas, quando suportadas pela plataforma.

Toda transferência será iniciada por ação explícita do usuário.

Não haverá envio automático de conteúdo.

# 12. Área de transferência

O Tomorrow Drop permitirá o envio manual do clipboard.

## 12.1 Conteúdos

- Texto.
- Links.
- Imagens, quando disponíveis pelas APIs da plataforma.

## 12.2 Privacidade

O aplicativo não deverá ler o clipboard automaticamente ao iniciar.

A leitura ocorrerá somente após o usuário selecionar **Enviar área de transferência**.

Antes do envio, será apresentada uma prévia do conteúdo.

## 12.3 Recebimento

O conteúdo recebido não substituirá automaticamente o clipboard do destinatário.

O usuário poderá copiar o texto recebido manualmente.

Não haverá sincronização contínua nem histórico persistente.

# 13. Autorização de recebimento

**Decisão aprovada:** toda transferência exige aceitação explícita do destinatário.

Isso também se aplica a dispositivos já confiáveis.

A solicitação deverá apresentar:

- Identidade autenticada do remetente.
- Tipo de conteúdo.
- Nome do arquivo, quando aplicável.
- Tamanho do arquivo ou conjunto de arquivos.
- Ações Aceitar e Recusar.

A recusa deverá encerrar a solicitação sem salvar o conteúdo.

Solicitações não respondidas terão timeout.

# 14. Destino dos arquivos

**Decisão aprovada:** Downloads será o destino padrão.

## 14.1 Desktop

Utilizar a pasta Downloads correspondente ao usuário do sistema operacional.

## 14.2 Android

Utilizar os mecanismos apropriados de armazenamento do Android, respeitando as permissões e APIs disponíveis.

## 14.3 iOS/iPadOS

Utilizar uma localização acessível pelo aplicativo Arquivos ou mecanismo nativo de exportação.

O aplicativo não deverá pressupor que exista uma pasta Downloads universalmente gravável como no desktop.

## 14.4 Conflitos

Arquivos existentes não poderão ser sobrescritos silenciosamente.

A implementação deverá adotar uma política segura de renomeação ou solicitar uma decisão explícita ao usuário.

# 15. Integridade e arquivos temporários

Arquivos recebidos serão gravados inicialmente em armazenamento temporário.

Ao concluir, o aplicativo deverá:

1. Verificar a conclusão do protocolo.
2. Validar o tamanho esperado.
3. Verificar a integridade do conteúdo conforme o mecanismo definido pelo protocolo.
4. Disponibilizar o arquivo definitivo.
5. Remover os dados temporários desnecessários.

A implementação deverá detectar truncamentos e falhas de transmissão.

Quando suportado, a disponibilização final deverá utilizar renomeação atômica.

Arquivos incompletos nunca poderão ser apresentados como concluídos.

# 16. Transferências interrompidas

**Decisão aprovada: não haverá retomada.**

Uma transferência interrompida será invalidada.

Uma nova tentativa deverá começar do primeiro byte.

Não haverá checkpoints, armazenamento persistente de progresso ou sincronização de estado entre tentativas.

São consideradas interrupções:

- Falha de rede.
- Cancelamento.
- Timeout.
- Encerramento do aplicativo.
- Mudança de estado incompatível com a transferência.

Arquivos parciais deverão ser removidos.

# 17. Ciclo de vida — Desktop

Aplica-se a Linux, Windows e macOS.

## 17.1 Aplicativo visível

Enquanto a janela estiver visível:

- A descoberta UDP poderá funcionar.
- O listener TCP/TLS estará disponível.
- Novas conexões poderão ser aceitas.
- Transferências poderão ser iniciadas e recebidas.

## 17.2 Aplicativo minimizado

Ao minimizar:

- Suspender consultas UDP.
- Fechar o listener de novas conexões.
- Não aceitar novas solicitações.
- Manter somente transferências já autorizadas e em andamento.
- Encerrar as conexões mantidas ao terminar, cancelar, falhar ou atingir timeout.

Se não houver transferência ativa, os recursos de rede serão liberados imediatamente.

## 17.3 Encerramento

Fechar explicitamente o aplicativo deverá:

- Cancelar transferências.
- Encerrar conexões.
- Fechar listeners.
- Interromper descoberta.
- Liberar recursos.

Minimizar e encerrar são ações distintas.

# 18. Ciclo de vida — Mobile

Aplica-se a Android e iOS/iPadOS.

O aplicativo deverá permanecer em primeiro plano durante:

- Descoberta.
- Pareamento.
- Envio.
- Recebimento.

Ao entrar em segundo plano:

- Interromper descoberta.
- Fechar listeners.
- Cancelar transferências.
- Encerrar conexões.
- Invalidar operações incompletas.
- Liberar recursos de rede.

Não haverá serviço de transferência em segundo plano no MVP.

Ao retornar ao primeiro plano, o aplicativo enviará uma nova consulta de descoberta imediatamente.

# 19. Interface gráfica

## 19.1 Diretrizes

A interface deverá ser:

- Minimalista.
- Clara.
- Consistente.
- Responsiva.
- Adequada a telas pequenas.
- Livre de elementos decorativos excessivos.
- Orientada a uma ação principal por tela.

Informações criptográficas detalhadas serão apresentadas principalmente durante pareamento, consulta de identidade e resolução de problemas de segurança.

## 19.2 Identidade visual

O logotipo utilizará **duas montanhas e um sol**.

O tema escuro será a referência visual inicial.

A paleta deverá privilegiar cores neutras, com acentos roxos discretos e indicadores de sucesso ou erro quando necessários.

## 19.3 Telas

### Início

- Enviar.
- Receber.
- Dispositivos encontrados.
- Atualizar.
- Conexão manual.
- Configurações.

### Pareamento

- Nome do dispositivo local.
- Nome do dispositivo remoto.
- Fingerprints completos.
- Instruções para comparação cruzada.
- Confirmar.
- Cancelar.

### Enviar

- Selecionar arquivos.
- Enviar clipboard.
- Escrever texto.

### Solicitação de recebimento

- Dispositivo remetente.
- Informações do conteúdo.
- Aceitar.
- Recusar.

### Transferência

- Nome do conteúdo.
- Progresso.
- Cancelar.

### Conclusão

- Confirmação.
- Abrir conteúdo.
- Voltar.

### Dispositivos confiáveis

- Lista de dispositivos.
- Fingerprint.
- Remover confiança.

### Configurações

- Identidade local.
- Preferências essenciais.
- Informações do aplicativo.

Não haverá telas de QR Code.

# 20. CLI

O aplicativo incluirá a CLI `tdrop`.

Comandos previstos:

- `tdrop discover`
- `tdrop identity show`
- `tdrop pair <destino>`
- `tdrop trust list`
- `tdrop trust remove <id>`
- `tdrop send <arquivo> --to <destino>`
- `tdrop send --stdin --to <destino>`
- `tdrop receive`
- `tdrop daemon`

A CLI deverá utilizar o mesmo núcleo da GUI.

## 20.1 Recebimento pela CLI

O recebimento exigirá autorização explícita.

Por padrão, a autorização será interativa.

Qualquer política não interativa deverá ser configurada conscientemente pelo operador e documentada, sem enfraquecer a autenticação de identidade.

## 20.2 Daemon

`tdrop daemon` permitirá operação contínua quando iniciado explicitamente.

Esse modo será uma exceção à política de disponibilidade vinculada à janela gráfica.

# 21. Armazenamento local e privacidade

O aplicativo poderá persistir:

- Chave privada local.
- Chaves públicas confiáveis.
- Identificadores e nomes amigáveis.
- Preferências essenciais.

O aplicativo não deverá persistir como histórico:

- Textos transferidos.
- Conteúdo do clipboard.
- Listas de arquivos enviados ou recebidos.
- Chaves de sessão.
- Progresso de transferências interrompidas.

Arquivos salvos intencionalmente pelo usuário permanecerão no destino escolhido.

Dados temporários deverão ser removidos após sua utilização ou durante a limpeza de sessões incompletas.

# 22. Modelo de ameaças

O Tomorrow Drop deverá proteger contra:

- Interceptação de tráfego.
- Modificação de dados em trânsito.
- Ataques MITM durante o primeiro contato, quando o usuário realiza corretamente a comparação fora de banda.
- Falsificação de anúncios UDP.
- Tentativas de transferência por dispositivos não autenticados.
- Alteração inesperada de chaves públicas confiáveis.
- Solicitações maliciosas de transferência.
- Abuso de recursos por mensagens ou conexões não autorizadas.

## 22.1 Limites de segurança

O projeto não prometerá proteção contra:

- Sistema operacional comprometido.
- Roubo da chave privada.
- Adulteração física ou lógica das telas.
- Usuário que confirme fingerprints sem compará-los.
- Negação de serviço na rede local.
- Bloqueios impostos pela infraestrutura de rede.

O estabelecimento de TLS, isoladamente, não será tratado como autenticação de identidade.

# 23. Plano de testes

## 23.1 Segurança

- Identidades Ed25519.
- Fingerprints SHA-256.
- Autenticação TLS.
- Confirmação mútua.
- Rejeição de MITM.
- Bloqueio de identidades alteradas.
- Revogação de confiança.
- Rejeição de transferências não autorizadas.

## 23.2 Descoberta

- Consulta imediata ao abrir.
- Consulta periódica de 30 segundos.
- Atualização manual.
- Reinício do contador após atualização.
- Limitação de frequência.
- Respostas UDP unicast.
- Suspensão durante transferência.
- Funcionamento em redes empresariais.
- Conexão manual por IP e porta.

## 23.3 Transferências

- Arquivos pequenos e grandes.
- Múltiplos arquivos.
- Textos e links.
- Clipboard.
- Aceitação e recusa.
- Cancelamento.
- Falhas de rede.
- Timeout.
- Reinício desde o primeiro byte.
- Integridade.
- Limpeza de temporários.
- Colisões de nomes.

## 23.4 Ciclo de vida

- Desktop visível.
- Desktop minimizado durante transferência.
- Desktop minimizado sem transferência.
- Encerramento explícito.
- Mobile em primeiro plano.
- Mobile em segundo plano.
- Bloqueio de tela.
- Retorno ao aplicativo.

## 23.5 Compatibilidade

Executar testes em:

- Linux.
- Windows.
- macOS.
- Android.
- iOS/iPadOS.

# 24. Diretrizes técnicas aprovadas

Todas as decisões de produto e arquitetura estão aprovadas para implementação.

## 24.1 Interface multiplataforma

- Go como linguagem principal.
- Fyne como framework gráfico.
- Núcleo compartilhado com CLI.
- Suporte às cinco plataformas definidas.

## 24.2 Segurança

- TLS 1.3.
- Ed25519.
- SHA-256.
- Fingerprints completos.
- Pareamento manual com confirmação mútua.
- Confiança persistente até revogação.
- Pinning de identidade.
- Rejeição de chaves alteradas.

## 24.3 Armazenamento

- Downloads como destino padrão, conforme os mecanismos de cada plataforma.
- Arquivos temporários.
- Verificação de integridade.
- Limpeza de arquivos incompletos.
- Sem retomada de transferências.

## 24.4 Descoberta

- UDP broadcast local.
- Busca imediata ao abrir.
- Busca imediata ao atualizar.
- Intervalo periódico de 30 segundos.
- Respostas preferencialmente por UDP unicast.
- Limitação de frequência.
- Atraso aleatório de resposta.
- Conexão manual por IP e porta.

## 24.5 Permissões e ciclo de vida

- Sem QR Code.
- Sem permissão de câmera.
- Clipboard acessado somente por ação explícita.
- Desktop mantém transferências autorizadas após minimizar.
- Mobile exige primeiro plano.
- Sem serviços residentes no mobile.

## 24.6 Distribuição e qualidade

- Licença MIT.
- Dependências externas mínimas.
- Testes de segurança.
- Testes de interoperabilidade.
- Empacotamento para cada plataforma.
- Revisão de segurança antes da publicação estável.

**Status:** aprovado.

Testes, integrações e revisões são atividades de execução e verificação, não decisões de produto pendentes.

# 25. Plano de implementação

## Fase 1 — Núcleo seguro em Go

Implementar:

1. Identidades Ed25519.
2. Fingerprints SHA-256.
3. TLS 1.3.
4. Pareamento manual.
5. Confiança persistente.
6. Descoberta UDP.
7. Conexão manual.
8. Transferência básica de arquivos e textos.

## Fase 2 — Desktop e CLI

Implementar:

1. Interface Fyne.
2. Descoberta a cada 30 segundos.
3. Atualização manual.
4. Clipboard.
5. Autorização de recebimento.
6. Controle de ciclo de vida desktop.
7. CLI compartilhando o núcleo.

## Fase 3 — Mobile

Implementar:

1. Android.
2. iOS/iPadOS.
3. Integração com armazenamento.
4. Clipboard sob ação explícita.
5. Controle de primeiro plano.
6. Testes entre desktop e mobile.

## Fase 4 — Estabilização

Executar:

1. Testes de segurança.
2. Revisão do protocolo.
3. Testes de interoperabilidade.
4. Revisão de dependências.
5. Documentação.
6. Empacotamento.
7. Publicação sob licença MIT.

# 26. Critérios de aceitação do MVP

O MVP será considerado concluído quando:

1. Dispositivos na mesma rede permitida puderem se descobrir por UDP.
2. A descoberta ocorrer imediatamente ao abrir ou atualizar.
3. As consultas periódicas ocorrerem a cada 30 segundos enquanto o aplicativo estiver disponível.
4. A conexão manual por IP e porta funcionar.
5. O primeiro pareamento exigir comparação cruzada de fingerprints completos.
6. A confiança persistir até revogação manual.
7. Identidades alteradas forem rejeitadas.
8. O transporte utilizar TLS 1.3 com autenticação da identidade esperada.
9. Toda transferência recebida exigir autorização.
10. Arquivos, textos e links puderem ser transferidos.
11. O clipboard puder ser enviado manualmente.
12. Arquivos recebidos forem salvos no destino apropriado.
13. Arquivos incompletos não forem apresentados como concluídos.
14. Transferências interrompidas recomeçarem do zero.
15. O desktop permitir concluir transferências já autorizadas após minimizar.
16. O mobile exigir o aplicativo em primeiro plano.
17. Não houver QR Code nem necessidade de câmera.
18. Não houver conta, nuvem ou servidor central.
19. A CLI e a GUI compartilharem o mesmo núcleo.
20. O aplicativo funcionar nas plataformas definidas.
21. O código-fonte e a documentação estiverem disponíveis sob licença MIT.

---

# 27. Estado final da especificação

**Versão:** 1.9  
**Decisões de produto:** aprovadas  
**Arquitetura:** aprovada  
**Segurança:** abordagem aprovada; implementação sujeita a testes e revisão  
**Descoberta:** UDP sob demanda, com intervalo de 30 segundos  
**Pareamento:** fingerprint completo, sem QR Code  
**Interface:** Go + Fyne  
**Licença:** MIT  
**Próxima etapa:** implementação

---

# Tomorrow Drop

**Transferiu. Usou. Pronto.**

Um aplicativo local, direto e minimalista para compartilhar conteúdo entre dispositivos, com criptografia, autenticação explícita e controle do usuário.