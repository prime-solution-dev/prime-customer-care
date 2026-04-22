package ticketService

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"prime-customer-care/internal/db"
	"prime-customer-care/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetTicketsRequest struct {
	CustomerCodes []string `json:"customer_codes"`
	Page          int      `json:"page"`
	PageSize      int      `json:"page_size"`
}

type GetTicketsResponse struct {
	ResponseCode int             `json:"response_code"`
	Message      string          `json:"message"`
	Page         int             `json:"page"`
	TotalPages   int             `json:"total_pages"`
	Total        int64           `json:"total"`
	Data         []models.Ticket `json:"data"`
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

	return GetTickets(gormx, req)
}

func GetTickets(gormx *gorm.DB, request GetTicketsRequest) (*GetTicketsResponse, error) {
	if gormx == nil {
		return nil, errors.New("database connection is nil")
	}

	page := request.Page
	if page <= 0 {
		page = 1
	}

	pageSize := request.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	res := GetTicketsResponse{
		ResponseCode: 200,
		Message:      "success",
		Page:         page,
		Data:         []models.Ticket{},
	}

	query := gormx.Model(&models.Ticket{})

	if len(request.CustomerCodes) > 0 {
		query = query.Where("customer_name IN ?", request.CustomerCodes)
	}

	var total int64
	if err := query.Debug().Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tickets: %v", err)
	}

	var tickets []models.Ticket
	if err := query.
		Order("create_date DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&tickets).Error; err != nil {
		return nil, fmt.Errorf("failed to get tickets: %v", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	res.Total = total
	res.TotalPages = totalPages
	res.Data = tickets

	return &res, nil
}
