# ocs-messaging
Testes com produtores e consumidores do kafka.

## Dependências
- Zookeeper e Kafka
- librdkafka-dev (necessário para compilar o driver `confluent-kafka-go`)
- [confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go)
## Instruções para montagem do ambiente de desenvolvimento (usando o docker)
### Kafka
Sugere-se usar o docker para rodar o Kafka, porém o kafka tem como dependência o zookeeper.
- Criar a rede do docker que será usada para comunicação entre kafka e zookeeper
```
$ sudo docker network create --subnet 172.16.0.0/24 kafka-net
```
- Baixar a imagem do zookeeper (dependência necessária para gerenciamento dos brokers do kafka)
```
$ docker pull confluentinc/cp-zookeeper
```
- Iniciar o container do zookeeper usando a rede `kafka-net`
```
$ docker run -d --network kafka-net --hostname zookeeper --name zookeeper -p 2181:2181 -e ZOOKEEPER_CLIENT_PORT=2181 -e ZOOKEEPER_TICK_TIME=2000 confluentinc/cp-zookeeper
```
- Baixar a image do kafka
```
$ docker pull confluentinc/cp-kafka
```
- Inciar o container do kafka, usando a rede do docker chamada `kafka-net`, com as seguintes variáveis de ambiente:
  - KAFKA_ZOOKEEPER_CONNECT: endereço para conexão com o zookeeper
  - KAFKA_ADVERTISED_LISTENERS: informe de como clientes devem se conectar ao kafka
  - KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: quantas réplicas um tópico terá
> OBS.: o parâmetro `-p 9092:9092` faz o bind da porta `9092` do container para a porta `9092` da interface de rede local.
```
$ docker run -d --network kafka-net --hostname kafka --name kafka -p 9092:9092 -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9092 -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 confluentinc/cp-kafka
```
### librdkafka-dev
#### Adicionar o repositório da confluent
- Obter chave pública do repositório
```
$ wget -qO - https://packages.confluent.io/deb/7.0/archive.key | sudo apt-key add -
```
- Adicionar o repositório no arquivo sources.list
```
$ sudo add-apt-repository "deb [arch=amd64] https://packages.confluent.io/deb/7.0 stable main"
$ sudo add-apt-repository "deb https://packages.confluent.io/clients/deb $(lsb_release -cs) main"
```
- Atualizar o apt-get
```
$ sudo apt-get update
```
#### Obter o pacote librdkafka-dev que contém a librdkafka
```
$ apt-get install librdkafka-dev
```
### confluent-kafka-go
Obter a lib da confluent
```
$ go get gopkg.in/confluentinc/confluent-kafka-go.v1/kafka
```
