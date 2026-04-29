package opportunityService

import (
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

func opportunityTicketPairKey(opportunityID uuid.UUID, ticketCode string) string {
	return opportunityID.String() + "|" + ticketCode
}

func CreateOpportunityTicketsRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req []CreateOpportunityTicketsRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, err
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return CreateOpportunityTickets(gormx, ctx, req)
}

func CreateOpportunityTickets(
	gormx *gorm.DB,
	ctx *gin.Context,
	req []CreateOpportunityTicketsRequest,
) (*CreateOpportunityTicketsResponse, error) {

	if gormx == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if len(req) == 0 {
		return nil, fmt.Errorf("request is empty")
	}

	conUserID, _ := ctx.Get("user")
	userID := ""
	if v, ok := conUserID.(string); ok {
		userID = v
	}

	now := time.Now()

	requestedPairs := make([]CreateOpportunityTicketsRequest, 0, len(req))
	seenPairs := make(map[string]struct{}, len(req))

	for i, item := range req {
		if item.OpportunityID == uuid.Nil {
			return nil, fmt.Errorf("item[%d]: opportunity_id is required", i)
		}

		ticketCode := strings.TrimSpace(item.TicketCode)
		if ticketCode == "" {
			return nil, fmt.Errorf("item[%d]: ticket_code is required", i)
		}

		pairKey := opportunityTicketPairKey(item.OpportunityID, ticketCode)

		if _, ok := seenPairs[pairKey]; ok {
			continue
		}
		seenPairs[pairKey] = struct{}{}

		requestedPairs = append(requestedPairs, CreateOpportunityTicketsRequest{
			OpportunityID: item.OpportunityID,
			TicketCode:    ticketCode,
		})
	}

	existingRows := make([]models.OpportunityTicket, 0)

	query := gormx
	for _, item := range requestedPairs {
		query = query.Or(
			"(opportunity_id = ? AND ticket_code = ?)",
			item.OpportunityID,
			item.TicketCode,
		)
	}

	if err := query.Find(&existingRows).Error; err != nil {
		return nil, fmt.Errorf("failed to check existing opportunity tickets: %v", err)
	}

	existingByPair := make(map[string]models.OpportunityTicket, len(existingRows))
	for _, row := range existingRows {
		existingByPair[opportunityTicketPairKey(row.OpportunityID, row.TicketCode)] = row
	}

	rows := make([]models.OpportunityTicket, 0, len(requestedPairs))
	createdByPair := make(map[string]models.OpportunityTicket, len(requestedPairs))

	for _, item := range requestedPairs {
		pairKey := opportunityTicketPairKey(item.OpportunityID, item.TicketCode)

		if _, ok := existingByPair[pairKey]; ok {
			continue
		}

		row := models.OpportunityTicket{
			ID:            uuid.New(),
			OpportunityID: item.OpportunityID,
			TicketCode:    item.TicketCode,
			CreateDate:    now,
			CreateBy:      userID,
			UpdateDate:    now,
			UpdateBy:      userID,
		}

		rows = append(rows, row)
		createdByPair[pairKey] = row
	}

	if len(rows) > 0 {
		if err := gormx.Create(&rows).Error; err != nil {
			return nil, fmt.Errorf("failed to create opportunity tickets: %v", err)
		}
	}

	opportunityTickets := make([]OpportunityTicket, 0, len(requestedPairs))

	for _, item := range requestedPairs {
		pairKey := opportunityTicketPairKey(item.OpportunityID, item.TicketCode)

		if row, ok := existingByPair[pairKey]; ok {
			opportunityTickets = append(opportunityTickets, OpportunityTicket{ID: row.ID})
			continue
		}

		if row, ok := createdByPair[pairKey]; ok {
			opportunityTickets = append(opportunityTickets, OpportunityTicket{ID: row.ID})
		}
	}

	res := CreateOpportunityTicketsResponse{
		ResponseCode:       200,
		Message:            "create opportunity tickets success",
		OpportunityTickets: opportunityTickets,
	}

	return &res, nil
}
