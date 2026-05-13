package healthService

import (
	"github.com/gin-gonic/gin"
)

type GetHealthResponse struct {
	Status string `json:"status"`
}

func GetHealthRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	return GetHealth()
}

func GetHealth() (*GetHealthResponse, error) {
	return &GetHealthResponse{
		Status: "ok",
	}, nil
}
