package opportunityService

import (
	"encoding/json"
	"fmt"
	"os"
	"prime-customer-care/internal/db"
	"prime-customer-care/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UpdateOpportunityRequest struct {
	ID              uuid.UUID `json:"id"`
	OpportunityCode *string   `json:"opportunity_code"`
	OpportunityName *string   `json:"opportunity_name"`
	CustomerName    *string   `json:"customer_name"`
	TicketType      *string   `json:"ticket_type"`
	Tel             *string   `json:"tel"`
	Email           *string   `json:"email"`
	IsFollowerUp    *bool     `json:"is_follower_up"`
	BrandCode       *string   `json:"brand_code"`
	CsStaff         *string   `json:"cs_staff"`
	ProductName     *string   `json:"product_name"`
	OrderCode       *string   `json:"order_code"`
	Status          *string   `json:"status"`
}

func UpdateOpportunitiesRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req []UpdateOpportunityRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_prime_customer_care"))
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

	for _, item := range req {
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

		if err := gormx.Model(&models.Opportunity{}).
			Where("id = ?", item.ID).
			Updates(updateMap).Error; err != nil {
			return nil, fmt.Errorf("failed to update opportunity %s: %w", item.ID, err)
		}

		var updatedOpportunity models.Opportunity
		if err := gormx.Model(&models.Opportunity{}).
			Select("id", "opportunity_code").
			Where("id = ?", item.ID).
			First(&updatedOpportunity).Error; err != nil {
			return nil, fmt.Errorf("failed to fetch opportunity %s: %w", item.ID, err)
		}

		opportunities = append(opportunities, Opportunity{
			ID:              updatedOpportunity.ID,
			OpportunityCode: updatedOpportunity.OpportunityCode,
		})
	}

	res := CreateOpportunityResponse{
		ResponseCode:  200,
		Message:       "update success",
		Opportunities: opportunities,
	}

	return &res, nil
}
