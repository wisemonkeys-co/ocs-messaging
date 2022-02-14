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
```
- Se inscrever em um canal chamado `foo`
```
$ sub foo 1
```
- Publicar uma mensagem em um canal chamado `foo` com 11 caracteres
```
$ pub foo 11
$ Hello World
```
