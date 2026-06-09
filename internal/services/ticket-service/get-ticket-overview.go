package ticketService

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"prime-customer-care/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GetTicketOverviewRequest struct {
	ID               []string   `json:"id"`
	TicketCode       []string   `json:"ticket_code"`
	TicketChannel    []string   `json:"ticket_channel"`
	BrandCode        []string   `json:"brand_code"`
	TicketType       []string   `json:"ticket_type"`
	IsFollowUp       *bool      `json:"is_follow_up"`
	IsMissing        *bool      `json:"is_missing"`
	Status           []string   `json:"status"`
	CreateDateFrom   *time.Time `json:"create_date_from"`
	CreateDateTo     *time.Time `json:"create_date_to"`
	Page             int        `json:"page"`
	PageSize         int        `json:"page_size"`
	TicketCodeLike   string     `json:"ticket_code_like"`
	TelLike          string     `json:"tel_like"`
	CustomerNameLike string     `json:"customer_name_like"`
	EmailLike        string     `json:"email_like"`
	CreateByLike     string     `json:"create_by_like"`
}

type TicketOverview struct {
	ID            uuid.UUID  `json:"id"`
	TicketCode    string     `json:"ticket_code"`
	TicketChannel string     `json:"ticket_channel"`
	CreateDate    time.Time  `json:"create_date"`
	Tel           string     `json:"tel"`
	CustomerName  string     `json:"customer_name"`
	Email         string     `json:"email"`
	BrandCode     *string    `json:"brand_code"`
	TicketType    *string    `json:"ticket_type"`
	IsFollowerUp  *bool      `json:"is_follow_up"`
	IsMissing     *bool      `json:"is_missing"`
	StartCall     *time.Time `json:"start_call"`
	EndCall       *time.Time `json:"end_call"`
	CreateBy      string     `json:"create_by"`
	Status        string     `json:"status"`
}

type GetTicketOverviewResponse struct {
	ResponseCode int              `json:"response_code"`
	Message      string           `json:"message"`
	Page         int              `json:"page"`
	TotalPages   int              `json:"total_pages"`
	Total        int64            `json:"total"`
	Tickets      []TicketOverview `json:"tickets"`
}

func GetTicketOverviewRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req GetTicketOverviewRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return GetTicketOverview(gormx, req)
}

func GetTicketOverview(gormx *gorm.DB, request GetTicketOverviewRequest) (*GetTicketOverviewResponse, error) {
	if gormx == nil {
		return nil, errors.New("database connection is nil")
	}

	usePagination := request.PageSize > 0

	res := GetTicketOverviewResponse{
		ResponseCode: 200,
		Message:      "success",
		Page:         request.Page,
		Tickets:      []TicketOverview{},
	}

	query := gormx.Table("(?) AS ticket_overview", getTicketOverviewBaseQuery(gormx))

	if len(request.ID) > 0 {
		query = query.Where("id IN ?", request.ID)
	}

	if len(request.TicketCode) > 0 {
		query = query.Where("ticket_code IN ?", request.TicketCode)
	}

	if len(request.TicketChannel) > 0 {
		query = query.Where("ticket_channel IN ?", request.TicketChannel)
	}

	if len(request.BrandCode) > 0 {
		query = query.Where("brand_code IN ?", request.BrandCode)
	}

	if len(request.TicketType) > 0 {
		query = query.Where("ticket_type IN ?", request.TicketType)
	}

	if request.IsFollowUp != nil {
		query = query.Where("is_follower_up = ?", *request.IsFollowUp)
	}

	if request.IsMissing != nil {
		query = query.Where("is_missing = ?", *request.IsMissing)
	}

	if len(request.Status) > 0 {
		query = query.Where("status IN ?", request.Status)
	}

	if request.CreateDateFrom != nil {
		query = query.Where("create_date >= ?", *request.CreateDateFrom)
	}

	if request.CreateDateTo != nil {
		query = query.Where("create_date <= ?", *request.CreateDateTo)
	}

	if request.TicketCodeLike != "" {
		query = query.Where("ticket_code ILIKE ?", likeKeyword(request.TicketCodeLike))
	}

	if request.TelLike != "" {
		query = query.Where("tel ILIKE ?", likeKeyword(request.TelLike))
	}

	if request.CustomerNameLike != "" {
		query = query.Where("customer_name ILIKE ?", likeKeyword(request.CustomerNameLike))
	}

	if request.EmailLike != "" {
		query = query.Where("email ILIKE ?", likeKeyword(request.EmailLike))
	}

	if request.CreateByLike != "" {
		query = query.Where("create_by ILIKE ?", likeKeyword(request.CreateByLike))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tickets: %v", err)
	}

	fetchQuery := query.
		Order(`
			CASE
				WHEN status = 'PENDING' THEN 1
				WHEN status = 'PROCESS' THEN 2
				WHEN is_follower_up = true THEN 3
				WHEN status = 'COMPLETED' THEN 4
				ELSE 999
			END
		`).
		Order("create_date DESC").Order("id DESC")
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

	var tickets []TicketOverview
	if err := fetchQuery.Find(&tickets).Error; err != nil {
		return nil, fmt.Errorf("failed to get tickets: %v", err)
	}

	res.Total = total
	res.TotalPages = totalPages
	res.Tickets = tickets

	return &res, nil
}

func likeKeyword(keyword string) string {
	return "%" + keyword + "%"
}

func getTicketOverviewBaseQuery(gormx *gorm.DB) *gorm.DB {
	return gormx.Raw(`
		SELECT
			id,
			ticket_code,
			ticket_channel,
			create_date,
			tel,
			customer_name,
			email,
			NULL AS brand_code,
			NULL AS ticket_type,
			NULL AS is_follower_up,
			is_missing,
			start_call,
			end_call,
			create_by,
			status
		FROM ticket
		WHERE status = 'PENDING'

		UNION ALL

		SELECT
			id,
			opportunity_code AS ticket_code,
			create_by_channel AS ticket_channel,
			create_date,
			tel,
			customer_name,
			email,
			brand_code,
			ticket_type,
			is_follower_up,
			NULL AS is_missing,
			NULL AS start_call,
			NULL AS end_call,
			create_by,
			status
		FROM opportunity
	`)
}
