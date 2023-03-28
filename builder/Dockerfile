FROM alpine AS downloader
RUN apk add curl
RUN curl -Lo go.tar.gz https://go.dev/dl/go1.19.5.linux-amd64.tar.gz

FROM ubuntu:18.04
RUN apt update && apt upgrade -y
RUN apt install -y build-essential libusb-1.0-0-dev pkg-config
RUN apt clean

RUN mkdir -p opt workdir

WORKDIR opt
COPY --from=downloader go.tar.gz .
RUN tar -xvf go.tar.gz

VOLUME /workdir
WORKDIR /workdir

ENTRYPOINT [ "/opt/go/bin/go" ]
