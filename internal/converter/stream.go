package converter

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"sync"

	"github.com/chromedp/cdproto/cdp"
	cdpio "github.com/chromedp/cdproto/io"
)

// decodeBuffers reaproveita os buffers de decodificação entre requisições,
// evitando alocar um buffer novo por pedaço em alto volume.
var decodeBuffers sync.Pool

func getBuffer(size int) []byte {
	if b, ok := decodeBuffers.Get().(*[]byte); ok && cap(*b) >= size {
		return (*b)[:size]
	}
	return make([]byte, size)
}

func putBuffer(b []byte) {
	decodeBuffers.Put(&b)
}

// copyStream lê o PDF do Chromium em pedaços (IO.read) e escreve direto em w.
func copyStream(ctx context.Context, handle cdpio.StreamHandle, chunk int, w io.Writer) error {
	defer func() { _ = cdpio.Close(handle).Do(ctx) }()

	buf := getBuffer(base64.StdEncoding.DecodedLen(chunk * 2))
	defer putBuffer(buf)

	for {
		var res cdpio.ReadReturns
		if err := cdp.Execute(ctx, cdpio.CommandRead, cdpio.Read(handle).WithSize(int64(chunk)), &res); err != nil {
			return fmt.Errorf("falha ao ler o stream do PDF: %w", err)
		}

		data := []byte(res.Data)
		if res.Base64encoded {
			n, err := base64.StdEncoding.Decode(buf[:cap(buf)], data)
			if err != nil {
				return fmt.Errorf("falha ao decodificar o PDF: %w", err)
			}
			data = buf[:n]
		}

		if len(data) > 0 {
			if _, err := w.Write(data); err != nil {
				return fmt.Errorf("falha ao escrever o PDF: %w", err)
			}
		}
		if res.EOF {
			return nil
		}
	}
}
