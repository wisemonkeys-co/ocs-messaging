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
### Criptografia e autenticação
Sugere-se o uso do `.yml` para testes com criptografia `ssl` e autenticação `sasl`
```yml
version: '2'
services:
  zookeeper:
    image: confluentinc/cp-zookeeper
    hostname: zookeeper
    container_name: zookeeper
    ports:
      - '2181:2181'
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
      ZOOKEEPER_SASL_ENABLED: 'false'
  kafka:
    image: confluentinc/cp-kafka
    container_name: kafka
    ports:
      - '9092:9092'
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_INTER_BROKER_LISTENER_NAME: SASL_SSL
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: SASL_SSL:SASL_SSL
      KAFKA_ADVERTISED_LISTENERS: SASL_SSL://localhost:9092
      KAFKA_LISTENERS: SASL_SSL://:9092
      ZOOKEEPER_SASL_ENABLED: 'false'
      KAFKA_SASL_ENABLED_MECHANISMS: PLAIN
      KAFKA_SASL_MECHANISM_INTER_BROKER_PROTOCOL: PLAIN
      KAFKA_SSL_KEYSTORE_FILENAME: 'kafka.server.keystore.jks'
      KAFKA_SSL_KEYSTORE_CREDENTIALS: 'keystore_cred'
      KAFKA_SSL_KEY_CREDENTIALS: 'keystore_cred'
      KAFKA_SSL_TRUSTSTORE_FILENAME: 'kafka.server.truststore.jks'
      KAFKA_SSL_TRUSTSTORE_CREDENTIALS: 'truststore_cred'
      KAFKA_SSL_CLIENT_AUTH: 'required'
      KAFKA_SSL_ENDPOINT_IDENTIFICATION_ALGORITHM: ''
      KAFKA_OPTS: '-Djava.security.auth.login.config=/etc/kafka/secrets/kafka_server_jaas.conf'
    volumes:
      - ./certs:/etc/kafka/secrets
    depends_on:
      - zookeeper

```
#### Chaves, certificados e arquivos
Para gerar os arquivos necessários, basta seguir os passos descritos neste [tutorial](https://docs.confluent.io/platform/current/security/security_tutorial.html).   
Exemplo:
- Gerar chaves e certificados
  ```
  $ keytool -keystore kafka.server.keystore.jks -alias localhost -keyalg RSA -validity 365 -genkey
  # keystore-pass: keystorePass
  ```
- Criar Certificate Authority (CA)
  ```
  $ openssl req -new -x509 -keyout ca-key -out ca-cert -days 365
  # PEM pass phrase: pemPass
  $ keytool -keystore kafka.client.truststore.jks -alias CARoot -importcert -file ca-cert
  # truststore-pass: truststorePass
  $ keytool -keystore kafka.server.truststore.jks -alias CARoot -importcert -file ca-cert
  # truststore-pass: truststorePass
  ```
- Assinar o certificado
  ```
  $ keytool -keystore kafka.server.keystore.jks -alias localhost -certreq -file cert-file
  $ openssl x509 -req -CA ca-cert -CAkey ca-key -in cert-file -out cert-signed -days 365 -CAcreateserial -passin pass:pemPass
  $ keytool -keystore kafka.server.keystore.jks -alias CARoot -importcert -file ca-cert
  $ keytool -keystore kafka.server.keystore.jks -alias localhost -importcert -file cert-signed
  ```
Os arquivos `keystore_cred` e `truststore_cred`, descritos no `yml`, devem conter, respectivamente, as senhas geradas para os arquivos `kafka.server.keystore.jks` e `kafka.client.truststore.jks` e devem estar disponíveis na mesma pasta.
### kafkacat
Sugere-se usar, como ferramenta de desenvolvimento, o kafkacat.   
Exemplos de uso:
```
# produzir
$ echo 'testKey:{"event":"foo", "data": "bar"}' | kafkacat -b "localhost:9092" -t ocs-messaging-message-to-handle -X security.protocol=sasl_ssl -X sasl.mechanisms=PLAIN -X sasl.username=admin  -X sasl.password=admin-secret -X enable.ssl.certificate.verification=false -t "ocs-messaging-message-to-handle" -K:

# consumir
$ kafkacat -b "localhost:9092" -t ocs-messaging-message-to-handle -X security.protocol=sasl_ssl -X sasl.mechanisms=PLAIN -X sasl.username=admin  -X sasl.password=admin-secret -X enable.ssl.certificate.verification=false -t "ocs-messaging-message-to-handle" -K:
```