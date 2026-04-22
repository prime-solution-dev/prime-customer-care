package ticketService

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

type UpdateTicketRequest struct {
	ID            uuid.UUID  `json:"id"`
	TicketChannel *string    `json:"ticket_channel"`
	CustomerName  *string    `json:"customer_name"`
	Tel           *string    `json:"tel"`
	Email         *string    `json:"email"`
	IsWrong       *bool      `json:"is_wrong"`
	IsMissing     *bool      `json:"is_missing"`
	StartCall     *time.Time `json:"start_call"`
	EndCall       *time.Time `json:"end_call"`
	Status        *string    `json:"status"`
}

func UpdateTicketsRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req []UpdateTicketRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_prime_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.CloseGORM(gormx)

	return UpdateTickets(gormx, ctx, req)
}

func UpdateTickets(gormx *gorm.DB, ctx *gin.Context, req []UpdateTicketRequest) (*CreateTicketsResponse, error) {
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
	tickets := make([]Ticket, 0, len(req))

	for _, item := range req {
		if item.ID == uuid.Nil {
			return nil, fmt.Errorf("ticket id is required")
		}

		updateMap := map[string]interface{}{
			"update_date": now,
			"update_by":   userID,
		}

		if item.TicketChannel != nil {
			updateMap["ticket_channel"] = *item.TicketChannel
		}
		if item.CustomerName != nil {
			updateMap["customer_name"] = *item.CustomerName
		}
		if item.Tel != nil {
			updateMap["tel"] = *item.Tel
		}
		if item.Email != nil {
			updateMap["email"] = *item.Email
		}
		if item.IsWrong != nil {
			updateMap["is_wrong"] = *item.IsWrong
		}
		if item.IsMissing != nil {
			updateMap["is_missing"] = *item.IsMissing
		}
		if item.StartCall != nil {
			updateMap["start_call"] = item.StartCall
		}
		if item.EndCall != nil {
			updateMap["end_call"] = item.EndCall
		}
		if item.Status != nil {
			updateMap["status"] = *item.Status
		}

		if item.StartCall != nil && item.EndCall != nil {
			if item.StartCall.After(*item.EndCall) {
				return nil, fmt.Errorf("start_call must be before end_call")
			}
		}

		if err := gormx.Model(&models.Ticket{}).
			Where("id = ?", item.ID).
			Updates(updateMap).Error; err != nil {
			return nil, fmt.Errorf("failed to update ticket %s: %w", item.ID, err)
		}

		tickets = append(tickets, Ticket{
			ID: item.ID,
		})
	}

	res := CreateTicketsResponse{
		ResponseCode: 200,
		Message:      "update success",
	}

	return &res, nil
}
