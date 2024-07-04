FROM ubuntu:jammy
LABEL maintainer "kevinlin <linjiang1205@qq.com>"
RUN apt-get update && apt-get install -y  curl jq tini python3 python3-pip 
RUN pip3 install yq

RUN         curl -L https://github.com/soulteary/webhook/releases/download/3.6.0/webhook_3.6.0_linux_amd64.tar.gz -o webhook-linux-amd64.tar.gz && \
            mkdir webhook-linux-amd64 && tar -xzf webhook-linux-amd64.tar.gz -C webhook-linux-amd64 && \
            mv webhook-linux-amd64/webhook /usr/local/bin/webhook && \
            rm -rf webhook-linux-amd64.tar.gz webhook-linux-amd64
WORKDIR     /config

EXPOSE 9000/tcp