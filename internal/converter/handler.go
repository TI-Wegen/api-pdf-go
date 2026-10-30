package converter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/TI-Wegen/api-pdf-go/internal/browser"
	"github.com/TI-Wegen/api-pdf-go/internal/config"
)

type Handler struct {
	pool *browser.Pool
	cfg  config.Config
	log  *slog.Logger
}

func NewHandler(pool *browser.Pool, cfg config.Config, log *slog.Logger) *Handler {
	return &Handler{pool: pool, cfg: cfg, log: log}
}

// HandleConvert mantém o contrato original: POST /CreatePdfFromUrl {"url": "..."}.
func (h *Handler) HandleConvert(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	siteURL, opts, err := parseURLRequest(r)
	if err != nil {
		h.log.Warn("requisição de URL inválida", "erro", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.convert(w, r, siteURL, "", opts)
}

// HandleConvertHTML: POST /CreatePdfFromHtml com JSON {"html": "..."} ou corpo text/html.
func (h *Handler) HandleConvertHTML(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxBodyBytes)
	html, opts, err := parseHTMLRequest(r)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "HTML excede o tamanho máximo permitido", http.StatusRequestEntityTooLarge)
			return
		}
		h.log.Warn("requisição de HTML inválida", "erro", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.convert(w, r, "", html, opts)
}

func (h *Handler) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "emAndamento": h.pool.InFlight()})
}

func (h *Handler) convert(w http.ResponseWriter, r *http.Request, siteURL, html string, opts PdfOptions) {
	start := time.Now()
	origem := "html"
	if siteURL != "" {
		origem = siteURL
	}

	tab, release, err := h.pool.Acquire(r.Context())
	if err != nil {
		h.fail(w, origem, err)
		return
	}
	defer release()

	ctx, cancel := context.WithTimeout(tab, h.cfg.RenderTimeout)
	defer cancel()
	stop := context.AfterFunc(r.Context(), cancel) // cliente desconectou: aborta a renderização
	defer stop()

	out := newPdfWriter(w, opts.FileName)
	if err := render(ctx, siteURL, html, opts, h.cfg.StreamChunkBytes, out); err != nil {
		if !out.started {
			h.fail(w, origem, err)
			return
		}
		// Parte do PDF já foi enviada: derruba a conexão para o cliente
		// não receber um arquivo truncado como se fosse válido.
		h.log.Error("falha no meio do envio do PDF", "origem", origem, "erro", err)
		panic(http.ErrAbortHandler)
	}

	h.log.Info("PDF gerado", "origem", origem, "bytes", out.written, "duracao_ms", time.Since(start).Milliseconds())
}
