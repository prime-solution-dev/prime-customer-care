package models

import (
	"time"

	"github.com/google/uuid"
)

type Opportunity struct {
	ID              uuid.UUID `json:"id"`
	OpportunityCode string    `json:"opportunity_code"`
	TicketType      string    `json:"ticket_type"`
	OpportunityName string    `json:"opportunity_name"`
	CustomerName    string    `json:"customer_name"`
	Tel             string    `json:"tel"`
	Email           string    `json:"email"`
	IsFollowerUp    bool      `json:"is_follower_up"`
	BrandCode       string    `json:"brand_code"`
	CsStaff         string    `json:"cs_staff"`
	ProductName     string    `json:"product_name"`
	OrderCode       string    `json:"order_code"`
	Status          string    `json:"status"`
	CreateDate      time.Time `json:"create_date"`
	CreateBy        string    `json:"create_by"`
	UpdateDate      time.Time `json:"update_date"`
	UpdateBy        string    `json:"update_by"`
}

func (Opportunity) TableName() string {
	return "opportunity"
}

type OpportunityTicket struct {
	ID            uuid.UUID `json:"id"`
	OpportunityID uuid.UUID `json:"opportunity_id"`
	TicketCode    string    `json:"ticket_code"`
	CreateDate    time.Time `json:"create_date"`
	CreateBy      string    `json:"create_by"`
	UpdateDate    time.Time `json:"update_date"`
	UpdateBy      string    `json:"update_by"`
}

func (OpportunityTicket) TableName() string {
	return "opportunity_ticket"
}

type OpportunityRemark struct {
	ID            uuid.UUID `json:"id"`
	OpportunityID uuid.UUID `json:"opportunity_id"`
	Remark        string    `json:"remark"`
	CsStaff       string    `json:"cs_staff"`
	CreateDate    time.Time `json:"create_date"`
	CreateBy      string    `json:"create_by"`
	UpdateDate    time.Time `json:"update_date"`
	UpdateBy      string    `json:"update_by"`
}

func (OpportunityRemark) TableName() string {
	return "opportunity_remark"
}

type OpportunityRemarkAttachment struct {
	ID                  uuid.UUID `json:"id"`
	OpportunityRemarkID uuid.UUID `json:"opportunity_remark_id"`
	FileName            string    `json:"file_name"`
	FilePath            string    `json:"file_path"`
	FileType            string    `json:"file_type"`
	FileSize            int64     `json:"file_size"`
	CreateDate          time.Time `json:"create_date"`
	CreateBy            string    `json:"create_by"`
	UpdateDate          time.Time `json:"update_date"`
	UpdateBy            string    `json:"update_by"`
}

func (OpportunityRemarkAttachment) TableName() string {

	return "opportunity_remark_attachment"

}
