package opportunityService

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"prime-customer-care/internal/db"
	"prime-customer-care/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GetOpportunitiesRequest struct {
	ID              []string   `json:"id"`
	OpportunityCode []string   `json:"opportunity_code"`
	TicketType      []string   `json:"ticket_type"`
	CustomerName    []string   `json:"customer_name"`
	Tel             []string   `json:"tel"`
	Email           []string   `json:"email"`
	BrandCode       []string   `json:"brand_code"`
	OrderCode       []string   `json:"order_code"`
	Status          []string   `json:"status"`
	IsFollowerUp    *bool      `json:"is_follower_up"`
	CreateDateFrom  *time.Time `json:"create_date_from"`
	CreateDateTo    *time.Time `json:"create_date_to"`
	Page            int        `json:"page"`
	PageSize        int        `json:"page_size"`
}

type OpportunityWithTickets struct {
	models.Opportunity
	Tickets []CreateOpportunityTicket          `json:"tickets"`
	Remarks []OpportunityRemarkWithAttachments `json:"remarks"`
}

type GetOpportunitiesResponse struct {
	ResponseCode  int                      `json:"response_code"`
	Message       string                   `json:"message"`
	Page          int                      `json:"page"`
	TotalPages    int                      `json:"total_pages"`
	Total         int64                    `json:"total"`
	Opportunities []OpportunityWithTickets `json:"opportunities"`
}

func GetOpportunitiesRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req GetOpportunitiesRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_prime_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return GetOpportunities(gormx, req)
}

func GetOpportunities(gormx *gorm.DB, request GetOpportunitiesRequest) (*GetOpportunitiesResponse, error) {
	if gormx == nil {
		return nil, errors.New("database connection is nil")
	}

	usePagination := request.PageSize > 0

	res := GetOpportunitiesResponse{
		ResponseCode:  200,
		Message:       "success",
		Page:          request.Page,
		Opportunities: []OpportunityWithTickets{},
	}

	query := gormx.Model(&models.Opportunity{})

	if len(request.ID) > 0 {
		query = query.Where("id IN ?", request.ID)
	}

	if len(request.OpportunityCode) > 0 {
		query = query.Where("opportunity_code IN ?", request.OpportunityCode)
	}

	if len(request.TicketType) > 0 {
		query = query.Where("ticket_type IN ?", request.TicketType)
	}

	if len(request.CustomerName) > 0 {
		query = query.Where("customer_name IN ?", request.CustomerName)
	}

	if len(request.Tel) > 0 {
		query = query.Where("tel IN ?", request.Tel)
	}

	if len(request.Email) > 0 {
		query = query.Where("email IN ?", request.Email)
	}

	if len(request.BrandCode) > 0 {
		query = query.Where("brand_code IN ?", request.BrandCode)
	}

	if len(request.OrderCode) > 0 {
		query = query.Where("order_code IN ?", request.OrderCode)
	}

	if len(request.Status) > 0 {
		query = query.Where("status IN ?", request.Status)
	}

	if request.IsFollowerUp != nil {
		query = query.Where("is_follower_up = ?", *request.IsFollowerUp)
	}

	if request.CreateDateFrom != nil {
		query = query.Where("create_date >= ?", *request.CreateDateFrom)
	}

	if request.CreateDateTo != nil {
		query = query.Where("create_date <= ?", *request.CreateDateTo)
	}

	var total int64
	if err := query.Debug().Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count opportunities: %v", err)
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

	var opportunities []models.Opportunity
	if err := fetchQuery.Find(&opportunities).Error; err != nil {
		return nil, fmt.Errorf("failed to get opportunities: %v", err)
	}

	opportunityIDs := make([]uuid.UUID, 0, len(opportunities))
	for _, opportunity := range opportunities {
		opportunityIDs = append(opportunityIDs, opportunity.ID)
	}

	ticketsByOpportunityID := make(map[uuid.UUID][]CreateOpportunityTicket, len(opportunities))
	remarksByOpportunityID := make(map[uuid.UUID][]OpportunityRemarkWithAttachments, len(opportunities))
	if len(opportunityIDs) > 0 {
		var opportunityTickets []models.OpportunityTicket
		if err := gormx.
			Model(&models.OpportunityTicket{}).
			Select("opportunity_id", "ticket_code").
			Where("opportunity_id IN ?", opportunityIDs).
			Order("ticket_code ASC").
			Find(&opportunityTickets).
			Error; err != nil {
			return nil, fmt.Errorf("failed to get opportunity tickets: %v", err)
		}

		for _, opportunityTicket := range opportunityTickets {
			ticketsByOpportunityID[opportunityTicket.OpportunityID] = append(
				ticketsByOpportunityID[opportunityTicket.OpportunityID],
				CreateOpportunityTicket{TicketCode: opportunityTicket.TicketCode},
			)
		}

		var remarks []models.OpportunityRemark
		if err := gormx.
			Model(&models.OpportunityRemark{}).
			Where("opportunity_id IN ?", opportunityIDs).
			Order("create_date DESC").
			Find(&remarks).
			Error; err != nil {
			return nil, fmt.Errorf("failed to get opportunity remarks: %v", err)
		}

		remarkIDs := make([]uuid.UUID, 0, len(remarks))
		for _, remark := range remarks {
			remarkIDs = append(remarkIDs, remark.ID)
		}

		attachmentMap := make(map[uuid.UUID][]models.OpportunityRemarkAttachment, len(remarks))
		if len(remarkIDs) > 0 {
			var attachments []models.OpportunityRemarkAttachment
			if err := gormx.
				Where("opportunity_remark_id IN ?", remarkIDs).
				Order("create_date ASC").
				Find(&attachments).Error; err != nil {
				return nil, fmt.Errorf("failed to get opportunity remark attachments: %v", err)
			}

			for _, attachment := range attachments {
				attachmentMap[attachment.OpportunityRemarkID] = append(
					attachmentMap[attachment.OpportunityRemarkID],
					attachment,
				)
			}
		}

		for _, remark := range remarks {
			remarksByOpportunityID[remark.OpportunityID] = append(
				remarksByOpportunityID[remark.OpportunityID],
				OpportunityRemarkWithAttachments{
					ID:            remark.ID,
					OpportunityID: remark.OpportunityID,
					Remark:        remark.Remark,
					CsStaff:       remark.CsStaff,
					CreateDate:    remark.CreateDate,
					CreateBy:      remark.CreateBy,
					UpdateDate:    remark.UpdateDate,
					UpdateBy:      remark.UpdateBy,
					Attachments:   attachmentMap[remark.ID],
				},
			)
		}
	}

	opportunitiesWithTickets := make([]OpportunityWithTickets, 0, len(opportunities))
	for _, opportunity := range opportunities {
		tickets := ticketsByOpportunityID[opportunity.ID]
		if tickets == nil {
			tickets = []CreateOpportunityTicket{}
		}
		remarks := remarksByOpportunityID[opportunity.ID]
		if remarks == nil {
			remarks = []OpportunityRemarkWithAttachments{}
		}

		opportunitiesWithTickets = append(opportunitiesWithTickets, OpportunityWithTickets{
			Opportunity: opportunity,
			Tickets:     tickets,
			Remarks:     remarks,
		})
	}

	res.Total = total
	res.TotalPages = totalPages
	res.Opportunities = opportunitiesWithTickets

	return &res, nil
}
