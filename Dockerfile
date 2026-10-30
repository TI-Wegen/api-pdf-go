FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app ./main.go

FROM debian:bookworm-slim

# tini como PID 1 recolhe os processos-filho do Chromium que terminam,
# evitando acúmulo de processos zumbis em execuções longas.
RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    tini \
    fonts-liberation \
    fonts-noto-color-emoji \
    ca-certificates \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

ENV CHROME_PATH=/usr/bin/chromium \
    MAX_CONCURRENT_PAGES=4 \
    BROWSER_RECYCLE_AFTER=200 \
    GOMEMLIMIT=256MiB

RUN useradd --create-home --shell /usr/sbin/nologin app
USER app
WORKDIR /home/app

COPY --from=builder /app/app .

EXPOSE 8080

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["./app"]
