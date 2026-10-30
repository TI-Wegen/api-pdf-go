package config

import (
	"os"
	"runtime"
	"strconv"
	"time"
)

// Config reúne as configurações da API, todas sobrescrevíveis por variável de ambiente.
type Config struct {
	Port string

	// MaxConcurrentPages limita quantas abas renderizam ao mesmo tempo.
	// É o principal controle de memória: cada aba custa dezenas a centenas de MB.
	MaxConcurrentPages int

	// RecycleAfter reinicia o Chromium após N renderizações, devolvendo ao SO
	// a memória que o navegador acumula (fragmentação, caches, vazamentos).
	RecycleAfter int

	// QueueTimeout é o tempo máximo que uma requisição espera por uma vaga.
	QueueTimeout time.Duration

	// RenderTimeout é o tempo máximo de uma renderização (carregar + imprimir).
	RenderTimeout time.Duration

	// MaxBodyBytes limita o tamanho do corpo (HTML) aceito.
	MaxBodyBytes int64

	// StreamChunkBytes é o tamanho de cada pedaço lido do PDF no Chromium.
	StreamChunkBytes int

	ChromePath string
}

func Load() Config {
	return Config{
		Port:               env("PORT", "8080"),
		MaxConcurrentPages: envInt("MAX_CONCURRENT_PAGES", max(2, runtime.NumCPU())),
		RecycleAfter:       envInt("BROWSER_RECYCLE_AFTER", 200),
		QueueTimeout:       envDuration("QUEUE_TIMEOUT", 30*time.Second),
		RenderTimeout:      envDuration("RENDER_TIMEOUT", 45*time.Second),
		MaxBodyBytes:       int64(envInt("MAX_BODY_BYTES", 10<<20)),
		StreamChunkBytes:   envInt("STREAM_CHUNK_BYTES", 256<<10),
		ChromePath:         os.Getenv("CHROME_PATH"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return fallback
}
