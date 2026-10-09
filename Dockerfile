FROM golang:1.26.9-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -mod=readonly -trimpath -ldflags='-s -w' -o /panaino-bot .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /panaino-bot /usr/local/bin/panaino-bot
USER 65534:65534
ENTRYPOINT ["/usr/local/bin/panaino-bot"]
WORKDIR /data
CMD ["--db", "/data/bot.db", "--config", "/data/config.json"]
