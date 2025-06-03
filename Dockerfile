FROM golang:1.24.3-bookworm as builder

WORKDIR /go/src

COPY go.mod ./
COPY go.sum ./
COPY main.go ./
COPY assets /go/src/assets
COPY cmd /go/src/cmd
COPY internal /go/src/internal
COPY logo /go/src/logo
COPY pkg /go/src/pkg
COPY web /go/src/web

RUN go build -o ./bin/mailslurper ./*.go

FROM debian:bookworm-slim

RUN apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y ca-certificates \
	&& update-ca-certificates

COPY config.yaml ~/.config/mailslurper/config.yaml
COPY --from=builder /go/src/bin/mailslurper /go/bin/mailslurper

ENV PATH="/go/bin:${PATH}"
ENTRYPOINT ["mailslurper all"]
