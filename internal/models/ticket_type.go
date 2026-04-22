package models

import (
	"time"

	"github.com/google/uuid"
)

type TicketType struct {
	ID        uuid.UUID `json:"id"`
	TypeCode  string    `json:"type_code"`
	TypeName  string    `json:"type_name"`
	CreateBy  *string   `json:"create_by"`
	CreateDtm time.Time `json:"create_dtm"`
	UpdateBy  *string   `json:"update_by"`
	UpdateDtm time.Time `json:"update_dtm"`
}

func (TicketType) TableName() string {
	return "ticket_type"
}
