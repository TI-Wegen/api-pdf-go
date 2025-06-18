package converter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type Request struct {
	URL string `json:"url"`
}

type Handler struct {
	log *slog.Logger
}

func NewHandler(log *slog.Logger) *Handler {
	return &Handler{
		log: log,
	}
}
func (h *Handler) HandleConvert(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("falha ao decodificar o corpo da requisição", "erro", err)
		http.Error(w, "corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	if _, err := url.ParseRequestURI(req.URL); err != nil {
		h.log.Warn("URL inválida recebida", "url", req.URL, "erro", err)
		http.Error(w, "URL fornecida é inválida", http.StatusBadRequest)
		return
	}

	h.log.Info("iniciando a conversão de URL para PDF", "url", req.URL)

	pdfBuffer, err := h.generatePDF(r.Context(), req.URL)
	if err != nil {
		h.log.Error("falha ao gerar o PDF", "url", req.URL, "erro", err)
		http.Error(w, "erro interno ao gerar o PDF", http.StatusInternalServerError)
		return
	}

	h.log.Info("conversão para PDF concluída com sucesso", "url", req.URL, "tamanho_bytes", len(pdfBuffer))

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="download.pdf"`)

	if _, err := w.Write(pdfBuffer); err != nil {
		h.log.Error("falha ao escrever a resposta em PDF", "erro", err)
	}
}
func (h *Handler) generatePDF(ctx context.Context, siteURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(h.log.Info))
	defer cancel()

	var pdfBuffer []byte
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(siteURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuffer = buf
			return nil
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("falha na execução das tarefas do chromedp: %w", err)
	}

	return pdfBuffer, nil
}
