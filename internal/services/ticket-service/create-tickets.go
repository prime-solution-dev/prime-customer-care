package ticketService

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

type CreateTicketRequest struct {
	TicketChannel string    `json:"ticket_channel"`
	CustomerName  string    `json:"customer_name"`
	Tel           string    `json:"tel"`
	Email         string    `json:"email"`
	IsWrong       bool      `json:"is_wrong"`
	IsMissing     bool      `json:"is_missing"`
	StartCall     time.Time `json:"start_call"`
	EndCall       time.Time `json:"end_call"`
	Status        string    `json:"status"`
}

type CreateTicketsResponse struct {
	ResponseCode int      `json:"response_code"`
	Message      string   `json:"message"`
	Tickets      []Ticket `json:"tickets"`
}

type Ticket struct {
	ID         uuid.UUID `json:"id"`
	TicketCode string    `json:"ticket_code"`
}

func CreateTicketsRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req []CreateTicketRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_prime_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return CreateTickets(gormx, ctx, req)
}

func CreateTickets(gormx *gorm.DB, ctx *gin.Context, req []CreateTicketRequest) (*CreateTicketsResponse, error) {
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

	rows := make([]models.Ticket, 0, len(req))
	for _, item := range req {
		now := time.Now()

		var startCall *time.Time
		if !item.StartCall.IsZero() {
			startCall = &item.StartCall
		}

		var endCall *time.Time
		if !item.EndCall.IsZero() {
			endCall = &item.EndCall
		}

		rows = append(rows, models.Ticket{
			ID:            uuid.New(),
			TicketCode:    uuid.New().String(),
			TicketChannel: item.TicketChannel,
			CustomerName:  item.CustomerName,
			Tel:           item.Tel,
			Email:         item.Email,
			IsWrong:       item.IsWrong,
			IsMissing:     item.IsMissing,
			StartCall:     startCall,
			EndCall:       endCall,
			Status:        item.Status,
			CreateDate:    now,
			CreateBy:      userID,
			UpdateDate:    now,
			UpdateBy:      userID,
		})
	}

	if err := gormx.Create(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to create ticket: %v", err)
	}

	tickets := make([]Ticket, 0, len(rows))
	for _, row := range rows {
		tickets = append(tickets, Ticket{
			ID:         row.ID,
			TicketCode: row.TicketCode,
		})
	}

	res := CreateTicketsResponse{
		ResponseCode: 200,
		Message:      "create success",
		Tickets:      tickets,
	}

	return &res, nil
}
