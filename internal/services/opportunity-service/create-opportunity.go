package opportunityService

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"prime-customer-care/internal/db"
	"prime-customer-care/internal/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateOpportunityTicket struct {
	TicketCode string `json:"ticket_code"`
}

type CreateOpportunityRequest struct {
	OpportunityCode string                    `json:"opportunity_code"`
	OpportunityName string                    `json:"opportunity_name"`
	CustomerName    string                    `json:"customer_name"`
	TicketType      string                    `json:"ticket_type"`
	Tickets         []CreateOpportunityTicket `json:"tickets"`
	Tel             string                    `json:"tel"`
	Email           string                    `json:"email"`
	IsFollowerUp    bool                      `json:"is_follower_up"`
	BrandCode       string                    `json:"brand_code"`
	CsStaff         string                    `json:"cs_staff"`
	ProductName     string                    `json:"product_name"`
	OrderCode       string                    `json:"order_code"`
	Status          string                    `json:"status"`
}

type CreateOpportunityResponse struct {
	ResponseCode  int           `json:"response_code"`
	Message       string        `json:"message"`
	Opportunities []Opportunity `json:"opportunities"`
}

type Opportunity struct {
	ID              uuid.UUID `json:"id"`
	OpportunityCode string    `json:"opportunity_code,omitempty"`
}

func CreateOpportunitiesRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req []CreateOpportunityRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_prime_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return CreateOpportunities(gormx, ctx, req)
}

func CreateOpportunities(gormx *gorm.DB, ctx *gin.Context, req []CreateOpportunityRequest) (*CreateOpportunityResponse, error) {
	if gormx == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if len(req) == 0 {
		return nil, fmt.Errorf("request is empty")
	}

	conUserID, _ := ctx.Get("user")
	userID := ""
	if conUserID != nil {
		userID = conUserID.(string)
	}

	opportunityRows := make([]models.Opportunity, 0, len(req))
	ticketReqs := make([]CreateOpportunityTicketsRequest, 0)

	for i, item := range req {
		now := time.Now()
		opportunityID := uuid.New()
		opportunityCode := strings.TrimSpace(item.OpportunityCode)
		if opportunityCode == "" {
			opportunityCode = uuid.New().String()
		}

		opportunityRows = append(opportunityRows, models.Opportunity{
			ID:              opportunityID,
			OpportunityCode: opportunityCode,
			OpportunityName: item.OpportunityName,
			TicketType:      item.TicketType,
			CustomerName:    item.CustomerName,
			Tel:             item.Tel,
			Email:           item.Email,
			IsFollowerUp:    item.IsFollowerUp,
			BrandCode:       item.BrandCode,
			CsStaff:         item.CsStaff,
			ProductName:     item.ProductName,
			OrderCode:       item.OrderCode,
			Status:          item.Status,
			CreateDate:      now,
			CreateBy:        userID,
			UpdateDate:      now,
			UpdateBy:        userID,
		})

		for j, t := range item.Tickets {
			ticketCode := strings.TrimSpace(t.TicketCode)
			if ticketCode == "" {
				return nil, fmt.Errorf("item[%d].tickets[%d]: ticket_code is required", i, j)
			}

			ticketReqs = append(ticketReqs, CreateOpportunityTicketsRequest{
				OpportunityID: opportunityID,
				TicketCode:    ticketCode,
			})
		}
	}

	if err := gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&opportunityRows).Error; err != nil {
			return fmt.Errorf("failed to create opportunity: %v", err)
		}

		if len(ticketReqs) > 0 {
			if _, err := CreateOpportunityTickets(tx, ctx, ticketReqs); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	opportunities := make([]Opportunity, 0, len(opportunityRows))
	for _, row := range opportunityRows {
		opportunities = append(opportunities, Opportunity{
			ID:              row.ID,
			OpportunityCode: row.OpportunityCode,
		})
	}

	res := CreateOpportunityResponse{
		ResponseCode:  200,
		Message:       "create success",
		Opportunities: opportunities,
	}

	return &res, nil
}
