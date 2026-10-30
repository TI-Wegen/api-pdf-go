package converter

import (
	"context"
	"io"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// waitAssetsJS espera imagens e fontes terminarem de carregar. Necessário
// principalmente no HTML injetado, que não passa pelo ciclo normal de navegação.
const waitAssetsJS = `(async () => {
  const pending = Array.from(document.images).filter(img => !img.complete);
  await Promise.all(pending.map(img => new Promise(done => {
    img.addEventListener('load', done, { once: true });
    img.addEventListener('error', done, { once: true });
  })));
  if (document.fonts) await document.fonts.ready;
  return true;
})()`

// loadAction devolve as ações que colocam o conteúdo na aba.
func loadAction(siteURL, html string) chromedp.Tasks {
	if html == "" {
		return chromedp.Tasks{chromedp.Navigate(siteURL)}
	}
	return chromedp.Tasks{
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			tree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(tree.Frame.ID, html).Do(ctx)
		}),
	}
}

// render carrega o conteúdo e escreve o PDF em w aos pedaços, sem nunca
// manter o arquivo inteiro na memória do Go.
func render(ctx context.Context, siteURL, html string, opts PdfOptions, chunk int, w io.Writer) error {
	var ready bool
	return chromedp.Run(ctx,
		loadAction(siteURL, html),
		chromedp.Evaluate(waitAssetsJS, &ready, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, stream, err := printParams(opts).Do(ctx)
			if err != nil {
				return err
			}
			return copyStream(ctx, stream, chunk, w)
		}),
	)
}

func printParams(opts PdfOptions) *page.PrintToPDFParams {
	background := true
	if opts.PrintBackground != nil {
		background = *opts.PrintBackground
	}
	p := page.PrintToPDF().
		WithPrintBackground(background).
		WithLandscape(opts.Landscape).
		WithPreferCSSPageSize(opts.PreferCSSPageSize).
		WithTransferMode(page.PrintToPDFTransferModeReturnAsStream)
	if opts.Scale >= 0.1 && opts.Scale <= 2 {
		p = p.WithScale(opts.Scale)
	}
	return p
}
