FROM golang:alpine AS builder

WORKDIR $GOPATH/src/github.com/jidckii/patroni-exporter/

# Create appuser
COPY . .

RUN go get -d -v

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags='-w -s -extldflags "-static"' -a \
      -o /go/bin/patroni-exporter .

FROM alpine:latest

ENV USER=appuser
ENV UID=1001

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

COPY --from=builder /go/bin/patroni-exporter /usr/local/bin/patroni-exporter

USER appuser:appuser

ENTRYPOINT ["/usr/local/bin/patroni-exporter"]
