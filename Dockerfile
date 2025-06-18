FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compila o binário
RUN go build -o pdfscraping

FROM mcr.microsoft.com/playwright:v1.52.0-jammy

ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

RUN apt-get update && apt-get install -y ca-certificates

COPY --from=builder /app/pdfscraping /app/pdfscraping

EXPOSE 8080

ENTRYPOINT ["/app/pdfscraping"]
