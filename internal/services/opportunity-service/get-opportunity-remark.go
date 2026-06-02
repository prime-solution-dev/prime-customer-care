package opportunityService

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"prime-customer-care/internal/db"
	"prime-customer-care/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GetOpportunityRemarksRequest struct {
	OpportunityID string `json:"opportunity_id"`
	Page          int    `json:"page"`
	PageSize      int    `json:"page_size"`
}

type GetOpportunityRemarksResponse struct {
	ResponseCode int                                `json:"response_code"`
	Message      string                             `json:"message"`
	Page         int                                `json:"page"`
	TotalPages   int                                `json:"total_pages"`
	Total        int64                              `json:"total"`
	Data         []OpportunityRemarkWithAttachments `json:"data"`
}

func GetOpportunityRemarksRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req GetOpportunityRemarksRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return GetOpportunityRemarks(gormx, req)
}

func GetOpportunityRemarks(gormx *gorm.DB, request GetOpportunityRemarksRequest) (*GetOpportunityRemarksResponse, error) {
	if gormx == nil {
		return nil, errors.New("database connection is nil")
	}

	page := request.Page
	if page <= 0 {
		page = 1
	}

	pageSize := request.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	res := GetOpportunityRemarksResponse{
		ResponseCode: 200,
		Message:      "success",
		Page:         page,
		Data:         []OpportunityRemarkWithAttachments{},
	}

	query := gormx.Model(&models.OpportunityRemark{})

	if strings.TrimSpace(request.OpportunityID) != "" {
		opportunityID, err := uuid.Parse(strings.TrimSpace(request.OpportunityID))
		if err != nil {
			return nil, fmt.Errorf("invalid opportunity_id: %v", err)
		}
		query = query.Where("opportunity_id = ?", opportunityID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count opportunity remarks: %v", err)
	}

	var remarks []models.OpportunityRemark
	if err := query.
		Order("create_date DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&remarks).Error; err != nil {
		return nil, fmt.Errorf("failed to get opportunity remarks: %v", err)
	}

	if len(remarks) == 0 {
		res.Total = total
		res.TotalPages = 0
		return &res, nil
	}

	remarkIDs := make([]uuid.UUID, 0, len(remarks))
	for _, remark := range remarks {
		remarkIDs = append(remarkIDs, remark.ID)
	}

	var attachments []models.OpportunityRemarkAttachment
	if err := gormx.
		Where("opportunity_remark_id IN ?", remarkIDs).
		Order("create_date ASC").
		Find(&attachments).Error; err != nil {
		return nil, fmt.Errorf("failed to get opportunity remark attachments: %v", err)
	}

	attachmentMap := make(map[uuid.UUID][]models.OpportunityRemarkAttachment)
	for _, attachment := range attachments {
		attachmentMap[attachment.OpportunityRemarkID] = append(
			attachmentMap[attachment.OpportunityRemarkID],
			attachment,
		)
	}

	data := make([]OpportunityRemarkWithAttachments, 0, len(remarks))
	for _, remark := range remarks {
		data = append(data, OpportunityRemarkWithAttachments{
			ID:            remark.ID,
			OpportunityID: remark.OpportunityID,
			Remark:        remark.Remark,
			CsStaff:       remark.CsStaff,
			CreateDate:    remark.CreateDate,
			CreateBy:      remark.CreateBy,
			UpdateDate:    remark.UpdateDate,
			UpdateBy:      remark.UpdateBy,
			Attachments:   attachmentMap[remark.ID],
		})
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	res.Total = total
	res.TotalPages = totalPages
	res.Data = data

	return &res, nil
}
