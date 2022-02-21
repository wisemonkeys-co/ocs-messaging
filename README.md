# ocs-messaging
Message framework for wise OCS

## Sinopse
Testes iniciais com a ferramenta [NATS](https://nats.io/)

## Dependências
- NATS
  - Rodar usando o docker
  ```
  # rodar container do nats mapeando porta: 4222 para clientes, 8222 para monitoramento e a 6222 para comunicação (roteamento) entre membros de cluster
  $ docker run -d --name nats-main -p 4222:4222 -p 6222:6222 -p 8222:8222 nats
  ```

## Modos de uso
A aplicação usa um canal de eventos (subject) chamado `foo`. É possível rodá-la nos seguintes modos: `pub` e `sub`.

### SUB
No modo `sub` a aplicação se inscreve para receber as notificações do subject `foo`. Todo evento postado nesse subject é capturado, como texto, e impresso no console.   
Para executar a aplicação neste modo, basta rodar o código, sem enviar nenhum argumento.   
Exemplo:
```
$ go run main.go
```

### PUB
No modo `pub` a aplicação é capaz de postar mensagens, digitadas no console, no subject `foo`.   
Para executar a aplicação no modo `pub` basta rodar o código passando a flag `pub`.   
Exemplo:
```
$ go run main.go pub
```

## Ferramentas adicionais
Recomenda-se o uso da ferramenta `telnet` para monitorar ou interagir com a aplicação via console.
- Conectar no servidor do NATS
```
$ telnet localhost 4222
# localhost - ip do servidor NATS
# 4222 - porta para clientes
```
- Se inscrever em um subject chamado `foo`
```
sub foo 1
# sub - comando para se inscrever em um subject
# foo - nome do subject
# 1 - identificador da inscrição (sid)
```
- Publicar uma mensagem em um subject chamado `foo` com 11 caracteres
```
pub foo 11
Hello World
# pub - comando para enviar uma mensagem em um subject
# foo - nome do subject
# 11 - número de caracteres da mensagem
# Hello World - mensagem
```

## Detalhes do protocolo usado pelo NATS
O NATS utiliza um protocolo de texto com dez comandos no total. São eles:
- INFO (servidor): envio de metadados para os clientes.
- CONNECT (cliente): envio de credenciais e metadados para o servidor.
- PUB (cliente): envia uma mensagem para um subject.
- SUB (cliente): se inscreve em um subject.
- UNSUB (cliente): se desinscreve de um subject.
- MSG (servidor): envia uma mensagem para um cliente.
- +OK (servidor): ack informando que o comando foi processado.
- PING, PONG (cliente e servidor): comando para verificar conectividade.
- -ERR (servidor): informa erros.

Cada linha do protocolo é delimitada pelos caracteres `\r\n`. Todos os comandos de um client são processados em ordem pelo NATS.

### CONNECT
Permite informar metadados para o servidor usando um json.   
Exemplo para desligar o ack enviado pelo servidor e o detalhamento de erros:
```
connect {"verbose": false, "pedant": false}
```

### PING, PONG
São mensagens trocadas, não entre publishers e subscribers, mas entre cliente e servidor do NATS. Em intervalos regulares (default dois minutos) o servidor do NATS envia um `PING` para um cliente. Essa mensagem deve ser respondida com um `PONG`. Caso o cliente fique sem responder, o servidor do NATS encerra a conexão.   
Um cliente também pode enviar um `PING`, para o servidor, que deverá ser respondido com um `PONG`. Essa abordagem é especialmente útil em casos de envio para o servidor de uma série de comandos (linhas) de uma vez. Como os comandos, do cliente, são processados em ordem, pelo servidor, caso seja enviado um `PING` como última instrução, o `PONG` devolvido pelo servidor indicará o fim do processamento das linhas recebidas.   
Exemplo:
```
# Cliente 1 publicando duas mensagens e um ping
pub foo 3
bar
pub foz 3
baz
ping
PONG
```

### PUB
Usado para publicar uma mensagem. Obrigatoriamente deve ser informado: o nome do subject no qual a mensagem será publicada e o número de caracteres da mensagem. Opcionalmente, pode ser informado um subject de resposta para a mensagem. A linha seguinte a execução do `pub` deve conter a mensagem que será enviada, respeitando o número de caracteres informado.   
Exemplo 1 - envio de mensagem:
```
pub foo 11
Hello World
```
Exemplo 2 - envio de requisição:
```
# Client 1 se inscreve no subject reply 
sub reply my-sub-id

# Client 2 se inscreve no subject request
sub request my-request-id

# Client 1 envia uma mensagem informando o subject de resposta
pub request reply 13
anybody home?

# Client 2 recebe a mensagem e o subject onde a resposta deve ser publicada
MSG request my-request-id reply 13
anybody home?
```
OBS: O envio de requisições informa um subject de resposta que está sendo escutado pelo publisher. Os clients que recebem a requisição não são obrigados a respondê-la.

### SUB
Comando usado, pelo cliente, para receber mensagens enviadas para um subject. Obrigatoriamente, esse comando deve conter: o nome do subject e o identificador da inscrição no subject (sid).   
Exemplo:
```
# Cliente 1 se inscreve no subject "foo" com o sid "first-foo"
sub foo first-sub

# Cliente 2 se inscreve no subject "foo.bar" com o sid "1"
sub foo.bar 1

# Cliente 3 se inscreve em qualquer subject que comece com "foo." com o sid "all-foo"
sub foo.> all-foo

# Cliente 4 se inscreve em qualquer subject que termine com ".bar" com o sid "all-dot-bar"
sub *.bar all-dot-bar
```
#### Queue Subscripions
Opcionalmente um cliente pode se inscrever em um subject como membro de um grupo. Neste cenário, o servidor do NATS, sempre, a cada mensagem vai eleger um dos membros do grupo para recebê-la.   
Exemplo:
```
# Cliente 1 se inscreve no subject "foo" usando o grupo "worker" e o sid "first-queue-consumer"
sub foo worker first-queue-consumer

# Cliente 2 se inscreve no subject "foo" usando o grupo "worker" e o sid "second-queue-consumer"
sub foo worker second-queue-consumer

# Cliente 3 publica duas mensagens no subject "foo"
pub foo 3
abc
pub foo 3
def
ping

# Resposta do servidor para o Client 3
PONG

# Cliente 2
MSG foo second-queue-consumer 3
abc

# Cliente 1
MSG foo first-queue-consumer 3
def
```
