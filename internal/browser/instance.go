package browser

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chromedp/chromedp"
)

// instance é um processo Chromium compartilhado por várias abas.
// Campos refs/uses/retired são protegidos pelo mutex do Pool.
type instance struct {
	browserCtx  context.Context
	cancelAlloc context.CancelFunc
	log         *slog.Logger

	refs    int  // abas em uso agora
	uses    int  // abas já abertas nesta instância
	retired bool // não recebe novas abas; fecha quando refs chegar a 0
}

func newInstance(chromePath string, log *slog.Logger) (*instance, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.NoSandbox, // o container é a fronteira de isolamento; o sandbox do Chrome não sobe sem privilégios extras

		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("hide-scrollbars", true),
	)
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	browserCtx, _ := chromedp.NewContext(allocCtx,
		chromedp.WithErrorf(func(f string, a ...any) { log.Warn("chromedp", "msg", fmt.Sprintf(f, a...)) }),
	)

	// Run sem ações apenas inicia o processo do navegador.
	if err := chromedp.Run(browserCtx); err != nil {
		cancelAlloc()
		return nil, fmt.Errorf("falha ao iniciar o Chromium: %w", err)
	}

	log.Info("Chromium iniciado")
	return &instance{browserCtx: browserCtx, cancelAlloc: cancelAlloc, log: log}, nil
}

// alive indica se o processo ainda está conectado (o chromedp cancela o
// contexto quando o navegador cai).
func (i *instance) alive() bool {
	return i.browserCtx.Err() == nil
}

// newTab abre uma aba em um browser context próprio (cookies, cache e storage
// isolados), descartado por completo quando a aba fecha.
func (i *instance) newTab() (context.Context, context.CancelFunc) {
	return chromedp.NewContext(i.browserCtx, chromedp.WithNewBrowserContext())
}

func (i *instance) close() {
	if err := chromedp.Cancel(i.browserCtx); err != nil && i.alive() {
		i.log.Warn("falha ao fechar o Chromium graciosamente", "erro", err)
	}
	i.cancelAlloc() // mata o processo, se necessário, e remove o diretório temporário
	i.log.Info("Chromium encerrado", "renderizacoes", i.uses)
}
