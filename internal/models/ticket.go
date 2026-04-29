package models

import (
	"time"

	"github.com/google/uuid"
)

type Ticket struct {
	ID            uuid.UUID  `json:"id"`
	TicketCode    string     `json:"ticket_code"`
	TicketChannel string     `json:"ticket_channel"` // INCOMING_CALL, OUTGOING_CALL, CHAT, EMAIL
	CustomerName  string     `json:"customer_name"`
	Tel           string     `json:"tel"`
	Email         string     `json:"email"`
	IsWrong       bool       `json:"is_wrong"`
	IsMissing     bool       `json:"is_missing"`
	StartCall     *time.Time `json:"start_call"`
	EndCall       *time.Time `json:"end_call"`
	Status        string     `json:"status"` // TEMP, PENDING, COMPLETED, CANCELLED, PROCESS
	CreateDate    time.Time  `json:"create_date"`
	CreateBy      string     `json:"create_by"`
	UpdateDate    time.Time  `json:"update_date"`
	UpdateBy      string     `json:"update_by"`
}

func (Ticket) TableName() string {
	return "ticket"
}
