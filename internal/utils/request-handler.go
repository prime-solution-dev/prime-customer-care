package utils

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type StatusCoder interface {
	error
	StatusCode() int
}

func ProcessRequest(c *gin.Context, serviceFunc func(*gin.Context, string) (interface{}, error)) {
	// อ่าน JSON payload
	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Restore request body so downstream handlers can still read form-data/multipart payloads.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))

	// เรียกใช้ service function
	response, err := serviceFunc(c, string(jsonData))
	if err != nil {
		// ถ้า error มี StatusCode() ใช้ status นั้น
		if scErr, ok := err.(StatusCoder); ok {
			c.JSON(scErr.StatusCode(), gin.H{"error": scErr.Error()})
			return
		}

		// default 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ส่ง response
	c.JSON(http.StatusOK, response)
}

// ProcessRequestWithBinarySupport handles both JSON and binary responses
func ProcessRequestWithBinarySupport(c *gin.Context, serviceFunc func(*gin.Context, string) (interface{}, error)) {
	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))

	response, err := serviceFunc(c, string(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If response is nil, it means the service has already written binary data to the context
	// Don't write JSON response in this case
	if response != nil {
		c.JSON(http.StatusOK, response)
	}
}
func ProcessRequestMultiPart(c *gin.Context, serviceFunc func(*gin.Context) (interface{}, error)) {
	form, err := c.MultipartForm()
	if err != nil {
		log.Println("Error parsing multipart form:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get multipart form: " + err.Error()})
		return
	}

	for fieldName, fileHeaders := range form.File {
		for _, fileHeader := range fileHeaders {
			log.Println("Field Name:", fieldName)
			log.Println("Uploaded File:", fileHeader.Filename)
		}
	}

	response, err := serviceFunc(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
