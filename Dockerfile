FROM golang:1.18 as modules

ADD go.mod go.sum /m/
RUN cd /m && go mod download

FROM golang:1.18 as builder
COPY --from=modules /go/pkg /go/pkg
RUN mkdir -p /web-link

ADD . /web-link

WORKDIR /web-link
#do things under unpriveleged 'user'
RUN useradd -u 10001 user
# Собираем бинарный файл GOARCH=arm64 pi | amd64 x86
# main - это результирующий exe файл в корне билдера
RUN GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
   go build -o /main ./cmd/web-link

RUN chown user /main


FROM alpine

COPY --from=builder /etc/passwd /etc/passwd
# because program creates file inside we need to use image alpine which has commands to
# we make special dir with user rights
# otherwise exec cant create anything - perm denied as it creates process under this user
RUN mkdir -p /myappdir
RUN chown user /myappdir
USER user
WORKDIR /myappdir

# exe file has to have its own name different from dir (ie web-link would fail)
COPY --from=builder /main /myappdir/main

CMD ["/myappdir/main"]

# docker build -t web-link .
# docker run -p 8000:8000 web-link
# docker ps
# exec -it c1a013d02149 /bin/bash
#
# build image for raspberry pi arm64: (cross to arm)
# docker buildx build --platform linux/arm64 -t web-link:arm64 .
# copy docker image  to raspberry system:
# save it in tar archive
# docker save web-link:arm64  > weblink-arm64.tar
# copy to pi via scp
# scp weblink-arm64.tar user@192.168.1.210:/home/user
# unpack tar to local image docker repo (on pi side)
# docker load -i weblink-arm64.tar
# check & run
# docker image ls | grep arm
# docker run -p 8000:8000 web-link:arm64
# docker run -e PORT=8000 -e REPO=postgres://postuser:postpassword@192.168.1.204:5432/a4 -p 8000:8000 web-link_10
