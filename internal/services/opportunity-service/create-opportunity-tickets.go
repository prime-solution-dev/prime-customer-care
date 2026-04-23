package opportunityService

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"prime-customer-care/internal/db"
	"prime-customer-care/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateOpportunityTicketsRequest struct {
	OpportunityID uuid.UUID `json:"opportunity_id"`
	TicketCode    string    `json:"ticket_code"`
}

type CreateOpportunityTicketsResponse struct {
	ResponseCode       int                 `json:"response_code"`
	Message            string              `json:"message"`
	OpportunityTickets []OpportunityTicket `json:"opportunity_tickets"`
}

type OpportunityTicket struct {
	ID uuid.UUID `json:"id"`
}

func CreateOpportunityTicketsRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req []CreateOpportunityTicketsRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_prime_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return CreateOpportunityTickets(gormx, ctx, req)
}

func CreateOpportunityTickets(gormx *gorm.DB, ctx *gin.Context, req []CreateOpportunityTicketsRequest) (*CreateOpportunityTicketsResponse, error) {
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
	rows := make([]models.OpportunityTicket, 0, len(req))
	for i, item := range req {
		if item.OpportunityID == uuid.Nil {
			return nil, fmt.Errorf("item[%d]: opportunity_id is required", i)
		}
		if item.TicketCode == "" {
			return nil, fmt.Errorf("item[%d]: ticket_code is required", i)
		}

		rows = append(rows, models.OpportunityTicket{
			ID:            uuid.New(),
			OpportunityID: item.OpportunityID,
			TicketCode:    item.TicketCode,
			CreateDate:    now,
			CreateBy:      userID,
			UpdateDate:    now,
			UpdateBy:      userID,
		})
	}

	if err := gormx.Create(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to create opportunity tickets: %v", err)
	}

	opportunityTickets := make([]OpportunityTicket, 0, len(rows))
	for _, row := range rows {
		opportunityTickets = append(opportunityTickets, OpportunityTicket{
			ID: row.ID,
		})
	}

	res := CreateOpportunityTicketsResponse{
		ResponseCode:       200,
		Message:            "create opportunity tickets success",
		OpportunityTickets: opportunityTickets,
	}

	return &res, nil
}
