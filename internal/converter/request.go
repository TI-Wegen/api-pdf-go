package converter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PdfOptions são opcionais; os padrões reproduzem o comportamento anterior.
type PdfOptions struct {
	Landscape         bool    `json:"landscape"`
	PrintBackground   *bool   `json:"printBackground"`
	PreferCSSPageSize bool    `json:"preferCssPageSize"`
	Scale             float64 `json:"scale"`
	FileName          string  `json:"fileName"`
}

type urlRequest struct {
	URL     string     `json:"url"`
	Options PdfOptions `json:"options"`
}

type htmlRequest struct {
	HTML    string     `json:"html"`
	Options PdfOptions `json:"options"`
}

func parseURLRequest(r *http.Request) (string, PdfOptions, error) {
	var req urlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", req.Options, errors.New("corpo da requisição inválido")
	}
	u, err := url.ParseRequestURI(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", req.Options, errors.New("URL fornecida é inválida (use http ou https)")
	}
	return req.URL, req.Options, nil
}

// parseHTMLRequest aceita JSON {"html": "...", "options": {...}} ou o HTML cru
// com Content-Type text/html (opções via query string).
func parseHTMLRequest(r *http.Request) (string, PdfOptions, error) {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))

	var req htmlRequest
	if mediaType == "text/html" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return "", req.Options, fmt.Errorf("falha ao ler o corpo da requisição: %w", err)
		}
		req.HTML = string(body)
		req.Options = optionsFromQuery(r.URL.Query())
	} else if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", req.Options, fmt.Errorf("corpo da requisição inválido: %w", err)
	}

	if strings.TrimSpace(req.HTML) == "" {
		return "", req.Options, errors.New("o campo html é obrigatório")
	}
	return req.HTML, req.Options, nil
}

func optionsFromQuery(q url.Values) PdfOptions {
	opts := PdfOptions{
		Landscape:         q.Get("landscape") == "true",
		PreferCSSPageSize: q.Get("preferCssPageSize") == "true",
		FileName:          q.Get("fileName"),
	}
	if v, err := strconv.ParseBool(q.Get("printBackground")); err == nil {
		opts.PrintBackground = &v
	}
	if v, err := strconv.ParseFloat(q.Get("scale"), 64); err == nil {
		opts.Scale = v
	}
	return opts
}
