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
A aplicação usa um canal de eventos chamado `foo`. É possível rodá-la nos seguintes modos: `pub` e `sub`.

### SUB
No modo `sub` a aplicação se inscreve para receber as notificações do canal `foo`. Todo evento postado nesse canal é capturado, como texto, e impresso no console.   
Para executar a aplicação neste modo, basta rodar o código, sem enviar nenhum argumento.   
Exemplo:
```
$ go run main.go
```

### PUB
No modo `pub` a aplicação é capaz de postar mensagens, digitadas no console, no canal `foo`.   
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
# localhost - ip do servidor nats
# 4222 - porta para clientes
```
- Se inscrever em um canal chamado `foo`
```
$ sub foo 1
# sub - comando para se inscrever em um subject
# foo - nome do subject
# 1 - identificador da inscrição
```
- Publicar uma mensagem em um canal chamado `foo` com 11 caracteres
```
$ pub foo 11
# pub - comando para enviar uma mensagem em um subject
# foo - nome do subject
# 11 - número de caracteres da mensagem
$ Hello World
# Hello World - mensagem
```

## Detalhes do protocolo usado pela Nats
A Nats utiliza um protocolo de texto com dez comandos no total. Cada linha do protocolo é delimitada pelos caracteres `\r\n`. São eles:
- INFO (servidor): envio de metadados para os clientes.
- CONNECT (cliente): envio de credenciais e metadados para o servidor.
- PUB (cliente): envia uma mensagem para um subject.
- SUB (cliente): se inscreve em um subject.
- UNSUB (cliente): se desinscreve de um subject.
- MSG (servidor): envia uma mensagem para um cliente.
- +OK (servidor): ack informando que o comando foi processado.
- PING, PONG (cliente e servidor): comando para verificar conectividade.
- -ERR (servidor): informa erros.
