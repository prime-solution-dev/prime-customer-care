package ticketService

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"prime-customer-care/internal/db"
	"prime-customer-care/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetTicketsRequest struct {
	ID             []string   `json:"id"`
	TicketCode     []string   `json:"ticket_code"`
	TicketChannel  []string   `json:"ticket_channel"`
	CreateDateFrom *time.Time `json:"create_date_from"`
	CreateDateTo   *time.Time `json:"create_date_to"`
	Page           int        `json:"page"`
	PageSize       int        `json:"page_size"`
}

type GetTicketsResponse struct {
	ResponseCode int             `json:"response_code"`
	Message      string          `json:"message"`
	Page         int             `json:"page"`
	TotalPages   int             `json:"total_pages"`
	Total        int64           `json:"total"`
	Tickets      []models.Ticket `json:"tickets"`
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

	usePagination := request.PageSize > 0

	res := GetTicketsResponse{
		ResponseCode: 200,
		Message:      "success",
		Page:         request.Page,
		Tickets:      []models.Ticket{},
	}

	query := gormx.Model(&models.Ticket{})

	if len(request.ID) > 0 {
		query = query.Where("id IN ?", request.ID)
	}

	if len(request.TicketCode) > 0 {
		query = query.Where("ticket_code IN ?", request.TicketCode)
	}

	if len(request.TicketChannel) > 0 {
		query = query.Where("ticket_channel IN ?", request.TicketChannel)
	}

	if request.CreateDateFrom != nil {
		query = query.Where("create_date >= ?", *request.CreateDateFrom)
	}

	if request.CreateDateTo != nil {
		query = query.Where("create_date <= ?", *request.CreateDateTo)
	}

	var total int64
	if err := query.Debug().Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tickets: %v", err)
	}

	fetchQuery := query.Order("create_date DESC").Order("id DESC")
	totalPages := 0

	if usePagination {
		page := request.Page
		if page <= 0 {
			page = 1
		}

		pageSize := request.PageSize
		if total > 0 {
			totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
		}

		res.Page = page
		res.Total = total
		res.TotalPages = totalPages

		if total == 0 || page > totalPages {
			return &res, nil
		}

		offset := (page - 1) * pageSize
		fetchQuery = fetchQuery.Limit(pageSize).Offset(offset)
	} else if total > 0 {
		res.Page = 1
		totalPages = 1
	}

	var tickets []models.Ticket
	if err := fetchQuery.Find(&tickets).Error; err != nil {
		return nil, fmt.Errorf("failed to get tickets: %v", err)
	}

	res.Total = total
	res.TotalPages = totalPages
	res.Tickets = tickets

	return &res, nil
}
