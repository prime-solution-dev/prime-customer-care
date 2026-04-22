package ticketService

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"prime-customer-care/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetTicketsRequest struct {
	CustomerCodes []string `json:"customer_codes"`
}

type GetTicketsResponse struct {
	ResponseCode int    `json:"response_code"`
	Message      string `json:"message"`
	Page         int    `json:"page"`
	TotalPages   int    `json:"total_pages"`
}

func GetTicketsRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req GetTicketsRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_prime_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return GetTickets(nil, GetTicketsRequest{})
}

func GetTickets(gormx *gorm.DB, request GetTicketsRequest) (*GetTicketsResponse, error) {
	res := GetTicketsResponse{}

	return &res, nil
}
