package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SetPdfRequest struct {
	URL string `json:"url"`
}

func main() {
	r := gin.Default()
	pdfService := NewPdfScrapingService()
	r.POST("/CreatePdfFromUrl", func(c *gin.Context) {
		var req SetPdfRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		pdfBytes, err := pdfService.GetPdfContent(req.URL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Data(http.StatusOK, "application/pdf", pdfBytes)
	})

	if err := r.Run(); err != nil {
		log.Fatal(err)
	}
}
