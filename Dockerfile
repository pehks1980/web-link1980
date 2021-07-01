FROM golang:1.16 as modules

ADD go.mod go.sum /m/
RUN cd /m && go mod download

FROM golang:1.16 as builder
COPY --from=modules /go/pkg /go/pkg
RUN mkdir -p /web-link

ADD . /web-link

WORKDIR /web-link
#do things under unpriveleged 'user'
RUN useradd -u 10001 user
# Собираем бинарный файл
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
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
