package converter

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
)

type Handler struct {
	log *slog.Logger
	svc *Service
}

func NewHandler(log *slog.Logger) *Handler {
	return &Handler{
		log: log,
		svc: NewService(log),
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

	pdfBuffer, err := h.svc.GeneratePDF(r.Context(), req.URL)
	if err != nil {
		h.log.Error("falha ao gerar o PDF", "url", req.URL, "erro", err)
		http.Error(w, "erro interno ao gerar o PDF", http.StatusInternalServerError)
		return
	}

	h.log.Info("conversão para PDF concluída com sucesso", "url", req.URL, "tamanho_bytes", len(pdfBuffer))

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="url-download.pdf"`)

	if _, err := w.Write(pdfBuffer); err != nil {
		h.log.Error("falha ao escrever a resposta em PDF", "erro", err)
	}
}

func (h *Handler) HandleConvertHTML(w http.ResponseWriter, r *http.Request) {
	var req HTMLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("falha ao decodificar HTML", "erro", err)
		http.Error(w, "corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	if req.HTML == "" {
		h.log.Warn("HTML vazio recebido")
		http.Error(w, "HTML não pode ser vazio", http.StatusBadRequest)
		return
	}

	h.log.Info("iniciando a conversão de HTML para PDF")

	pdfBuffer, err := h.svc.GeneratePDFFromHTML(r.Context(), req.HTML)
	if err != nil {
		h.log.Error("falha ao gerar o PDF", "erro", err)
		http.Error(w, "erro interno ao gerar o PDF", http.StatusInternalServerError)
		return
	}

	base64Str := encodeToBase64(pdfBuffer)
	if base64Str != "" {
		h.log.Info("PDF gerado com sucesso e convertido em base64")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"pdf_base64": base64Str,
		})
		return
	}

	h.log.Warn("falha ao converter em base64, retornando PDF como bytes")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="html-download.pdf"`)
	w.Write(pdfBuffer)
}
