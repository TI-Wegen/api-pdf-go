package main

import (
	"errors"
	"os"

	"github.com/playwright-community/playwright-go"
)

type IPdfScraping interface {
	GetPdfContent(url string) ([]byte, error)
}

type PdfScrapingService struct{}

func NewPdfScrapingService() IPdfScraping {
	return &PdfScrapingService{}
}

func (p *PdfScrapingService) GetPdfContent(url string) ([]byte, error) {
	os.Setenv("PLAYWRIGHT_BROWSERS_PATH", "/app/bin/.playwright")

	if err := playwright.Install(); err != nil {
		return nil, errors.New("failed to install Playwright: " + err.Error())
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, err
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: nil,
	})

	if err != nil {
		return nil, err
	}

	page, err := context.NewPage()
	if err != nil {
		return nil, err
	}

	_, err = page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(60000),
	})
	if err != nil {
		return nil, err
	}

	pdf, err := page.PDF(playwright.PagePdfOptions{
		Format:          playwright.String("A4"),
		Landscape:       playwright.Bool(false),
		PrintBackground: playwright.Bool(true),
		Width:           playwright.String("100%"),
		Height:          playwright.String("100%"),
	})

	if err != nil {
		return nil, err
	}

	return pdf, nil
}
