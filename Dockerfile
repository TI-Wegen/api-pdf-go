# Etapa 1: Compilar o binário Go
FROM golang:1.24 AS builder


WORKDIR /app

# Copia arquivos necessários
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compila o binário
RUN go build -o pdfscraping

# Etapa 2: Imagem final com Playwright
FROM mcr.microsoft.com/playwright:v1.44.1-jammy

# Define path dos browsers (Playwright espera isso)
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

# Adiciona certificado, caso precise
RUN apt-get update && apt-get install -y ca-certificates

# Copia o binário da etapa anterior
COPY --from=builder /app/pdfscraping /app/pdfscraping

# Expõe a porta (ajuste conforme porta usada no Gin)
EXPOSE 8080

# Comando principal
ENTRYPOINT ["/app/pdfscraping"]
