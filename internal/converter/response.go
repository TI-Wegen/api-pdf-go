package converter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/TI-Wegen/api-pdf-go/internal/browser"
)

// pdfWriter só envia os cabeçalhos no primeiro byte do PDF, permitindo
// responder com erro HTTP normal se a falha acontecer antes disso.
type pdfWriter struct {
	w        http.ResponseWriter
	fileName string
	started  bool
	written  int64
}

func newPdfWriter(w http.ResponseWriter, fileName string) *pdfWriter {
	name := strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(fileName)
	if name == "" {
		name = "download.pdf"
	}
	return &pdfWriter{w: w, fileName: name}
}

func (p *pdfWriter) Write(b []byte) (int, error) {
	if !p.started {
		p.started = true
		p.w.Header().Set("Content-Type", "application/pdf")
		p.w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, p.fileName))
		p.w.WriteHeader(http.StatusOK)
	}
	n, err := p.w.Write(b)
	p.written += int64(n)
	return n, err
}

func (h *Handler) fail(w http.ResponseWriter, origem string, err error) {
	switch {
	case errors.Is(err, browser.ErrBusy):
		h.log.Warn("fila cheia, requisição recusada", "origem", origem)
		w.Header().Set("Retry-After", "5")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	case errors.Is(err, context.DeadlineExceeded):
		h.log.Error("tempo esgotado ao gerar o PDF", "origem", origem)
		http.Error(w, "tempo esgotado ao gerar o PDF", http.StatusGatewayTimeout)
	case errors.Is(err, context.Canceled):
		h.log.Warn("cliente cancelou a requisição", "origem", origem)
	default:
		h.log.Error("falha ao gerar o PDF", "origem", origem, "erro", err)
		http.Error(w, "erro interno ao gerar o PDF", http.StatusInternalServerError)
	}
}
