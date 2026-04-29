package opportunityService

import (
	"encoding/json"
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

type UpdateOpportunityRequest struct {
	ID              uuid.UUID                  `json:"id"`
	OpportunityCode *string                    `json:"opportunity_code"`
	OpportunityName *string                    `json:"opportunity_name"`
	CustomerName    *string                    `json:"customer_name"`
	TicketType      *string                    `json:"ticket_type"`
	Tickets         *[]CreateOpportunityTicket `json:"tickets"`
	Tel             *string                    `json:"tel"`
	Email           *string                    `json:"email"`
	IsFollowerUp    *bool                      `json:"is_follower_up"`
	BrandCode       *string                    `json:"brand_code"`
	CsStaff         *string                    `json:"cs_staff"`
	ProductName     *string                    `json:"product_name"`
	OrderCode       *string                    `json:"order_code"`
	Status          *string                    `json:"status"`
}

type updateOpportunityOperation struct {
	ID         uuid.UUID
	UpdateMap  map[string]interface{}
	TicketReqs []CreateOpportunityTicketsRequest
}

func UpdateOpportunitiesRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req []UpdateOpportunityRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.CloseGORM(gormx)

	return UpdateOpportunities(gormx, ctx, req)
}

func UpdateOpportunities(gormx *gorm.DB, ctx *gin.Context, req []UpdateOpportunityRequest) (*CreateOpportunityResponse, error) {
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

	now := time.Now()
	opportunities := make([]Opportunity, 0, len(req))
	operations := make([]updateOpportunityOperation, 0, len(req))

	for i, item := range req {
		if item.ID == uuid.Nil {
			return nil, fmt.Errorf("opportunity id is required")
		}

		updateMap := map[string]interface{}{
			"update_date": now,
			"update_by":   userID,
		}

		if item.OpportunityCode != nil {
			updateMap["opportunity_code"] = *item.OpportunityCode
		}
		if item.OpportunityName != nil {
			updateMap["opportunity_name"] = *item.OpportunityName
		}
		if item.CustomerName != nil {
			updateMap["customer_name"] = *item.CustomerName
		}
		if item.TicketType != nil {
			updateMap["ticket_type"] = *item.TicketType
		}
		if item.Tel != nil {
			updateMap["tel"] = *item.Tel
		}
		if item.Email != nil {
			updateMap["email"] = *item.Email
		}
		if item.IsFollowerUp != nil {
			updateMap["is_follower_up"] = *item.IsFollowerUp
		}
		if item.BrandCode != nil {
			updateMap["brand_code"] = *item.BrandCode
		}
		if item.CsStaff != nil {
			updateMap["cs_staff"] = *item.CsStaff
		}
		if item.ProductName != nil {
			updateMap["product_name"] = *item.ProductName
		}
		if item.OrderCode != nil {
			updateMap["order_code"] = *item.OrderCode
		}
		if item.Status != nil {
			updateMap["status"] = *item.Status
		}

		ticketReqs := make([]CreateOpportunityTicketsRequest, 0)
		if item.Tickets != nil && len(*item.Tickets) > 0 {
			ticketReqs = make([]CreateOpportunityTicketsRequest, 0, len(*item.Tickets))
			for j, ticket := range *item.Tickets {
				ticketCode := strings.TrimSpace(ticket.TicketCode)
				if ticketCode == "" {
					return nil, fmt.Errorf("item[%d].tickets[%d]: ticket_code is required", i, j)
				}

				ticketReqs = append(ticketReqs, CreateOpportunityTicketsRequest{
					OpportunityID: item.ID,
					TicketCode:    ticketCode,
				})
			}
		}

		operations = append(operations, updateOpportunityOperation{
			ID:         item.ID,
			UpdateMap:  updateMap,
			TicketReqs: ticketReqs,
		})
	}

	if err := gormx.Transaction(func(tx *gorm.DB) error {
		for _, operation := range operations {
			result := tx.Model(&models.Opportunity{}).
				Where("id = ?", operation.ID).
				Updates(operation.UpdateMap)
			if result.Error != nil {
				return fmt.Errorf("failed to update opportunity %s: %w", operation.ID, result.Error)
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("opportunity %s not found", operation.ID)
			}

			if len(operation.TicketReqs) > 0 {
				if _, err := CreateOpportunityTickets(tx, ctx, operation.TicketReqs); err != nil {
					return err
				}
			}

			opportunities = append(opportunities, Opportunity{
				ID: operation.ID,
			})
		}

		return nil
	}); err != nil {
		return nil, err
	}

	res := CreateOpportunityResponse{
		ResponseCode:  200,
		Message:       "update success",
		Opportunities: opportunities,
	}

	return &res, nil
}
