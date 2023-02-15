FROM golang:1.18.2-bullseye
LABEL MANTAINER=paulo.whyte@wisemonkeys.co

RUN mkdir -p /etc/app.d  && \
    mv /etc/localtime /etc/localtime.old && \
    ln -s /usr/share/zoneinfo/America/Sao_Paulo /etc/localtime && \
    apt-get install tzdata && \
    apt-get update && \
    apt-get install -y --no-install-recommends iputils-ping curl telnet ldnsutils dnsutils wget netcat && \
    mkdir -p /minu/log  && \ 
    adduser appuser 


ADD entrypoint.sh /entrypoint.sh

WORKDIR /go/src
COPY go.mod .
COPY go.sum .
RUN go mod download

RUN apt-get install librdkafka-dev -y

WORKDIR /go/src/ocs-messaging
ADD . .
RUN go build -o /minu/ocs-messaging . && \
    rm -r /go/src/ocs-messaging && \
    chown -R appuser /minu && \
    chown appuser /entrypoint.sh

WORKDIR /

USER appuser

ENTRYPOINT ["./entrypoint.sh"]
CMD ["start"]
