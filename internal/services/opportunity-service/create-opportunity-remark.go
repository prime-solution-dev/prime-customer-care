package opportunityService

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"prime-customer-care/internal/db"
	"prime-customer-care/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OpportunityRemarkWithAttachments struct {
	ID            uuid.UUID                            `json:"id"`
	OpportunityID uuid.UUID                            `json:"opportunity_id"`
	Remark        string                               `json:"remark"`
	CsStaff       string                               `json:"cs_staff"`
	CreateDate    time.Time                            `json:"create_date"`
	CreateBy      string                               `json:"create_by"`
	UpdateDate    time.Time                            `json:"update_date"`
	UpdateBy      string                               `json:"update_by"`
	Attachments   []models.OpportunityRemarkAttachment `json:"attachments"`
}

type CreateOpportunityRemarkResponse struct {
	ResponseCode int                              `json:"response_code"`
	Message      string                           `json:"message"`
	Data         OpportunityRemarkWithAttachments `json:"data"`
}

type CreateOpportunityRemarkInput struct {
	OpportunityID uuid.UUID
	Remark        string
	CsStaff       string
	CreateBy      string
	FileHeaders   []*multipart.FileHeader
}

func CreateOpportunityRemarkRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_prime_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return CreateOpportunityRemark(gormx, ctx)
}

func CreateOpportunityRemark(gormx *gorm.DB, ctx *gin.Context) (*CreateOpportunityRemarkResponse, error) {
	if gormx == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	opportunityIDStr := strings.TrimSpace(ctx.PostForm("opportunity_id"))
	remark := strings.TrimSpace(ctx.PostForm("remark"))
	csStaff := strings.TrimSpace(ctx.PostForm("cs_staff"))
	createBy := strings.TrimSpace(ctx.PostForm("create_by"))

	if opportunityIDStr == "" {
		return nil, fmt.Errorf("opportunity_id is required")
	}

	opportunityID, err := uuid.Parse(opportunityIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid opportunity_id: %v", err)
	}

	if createBy == "" {
		createBy = csStaff
	}
	if createBy == "" {
		createBy = "SYSTEM"
	}

	row := models.OpportunityRemark{}

	attachments := []models.OpportunityRemarkAttachment{}

	err = gormx.Transaction(func(tx *gorm.DB) error {
		form, _ := ctx.MultipartForm()
		fileHeaders := []*multipart.FileHeader{}
		if form != nil {
			fileHeaders = form.File["files"]
			if len(fileHeaders) == 0 {
				fileHeaders = form.File["file"]
			}
		}

		resRow, resAttachments, err := createOpportunityRemarkWithFiles(tx, ctx, CreateOpportunityRemarkInput{
			OpportunityID: opportunityID,
			Remark:        remark,
			CsStaff:       csStaff,
			CreateBy:      createBy,
			FileHeaders:   fileHeaders,
		})
		if err != nil {
			return err
		}

		row = resRow
		attachments = resAttachments
		return nil
	})
	if err != nil {
		return nil, err
	}

	res := CreateOpportunityRemarkResponse{
		ResponseCode: 200,
		Message:      "success",
		Data: OpportunityRemarkWithAttachments{
			ID:            row.ID,
			OpportunityID: row.OpportunityID,
			Remark:        row.Remark,
			CsStaff:       row.CsStaff,
			CreateDate:    row.CreateDate,
			CreateBy:      row.CreateBy,
			UpdateDate:    row.UpdateDate,
			UpdateBy:      row.UpdateBy,
			Attachments:   attachments,
		},
	}

	return &res, nil
}

func createOpportunityRemarkWithFiles(tx *gorm.DB, ctx *gin.Context, input CreateOpportunityRemarkInput) (models.OpportunityRemark, []models.OpportunityRemarkAttachment, error) {
	now := time.Now()
	remarkID := uuid.New()

	createBy := strings.TrimSpace(input.CreateBy)
	csStaff := strings.TrimSpace(input.CsStaff)
	if createBy == "" {
		createBy = csStaff
	}
	if createBy == "" {
		createBy = "SYSTEM"
	}

	basePath := strings.TrimSpace(os.Getenv("path_attachment"))
	if basePath == "" {
		basePath = "./attachment"
	}

	row := models.OpportunityRemark{
		ID:            remarkID,
		OpportunityID: input.OpportunityID,
		Remark:        strings.TrimSpace(input.Remark),
		CsStaff:       csStaff,
		CreateDate:    now,
		CreateBy:      createBy,
		UpdateDate:    now,
		UpdateBy:      createBy,
	}

	if err := tx.Create(&row).Error; err != nil {
		return models.OpportunityRemark{}, nil, fmt.Errorf("failed to create opportunity remark: %v", err)
	}

	if len(input.FileHeaders) == 0 {
		return row, []models.OpportunityRemarkAttachment{}, nil
	}

	saveDir := filepath.Join(basePath, "opportunity_remark", remarkID.String())
	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		return models.OpportunityRemark{}, nil, fmt.Errorf("failed to create directory: %v", err)
	}

	allowedExt := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".xls":  true,
		".xlsx": true,
	}

	attachments := make([]models.OpportunityRemarkAttachment, 0, len(input.FileHeaders))
	for _, fileHeader := range input.FileHeaders {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if !allowedExt[ext] {
			return models.OpportunityRemark{}, nil, fmt.Errorf("file type not allowed: %s", ext)
		}

		if fileHeader.Size > 10*1024*1024 {
			return models.OpportunityRemark{}, nil, fmt.Errorf("file size exceeds 10MB: %s", fileHeader.Filename)
		}

		attachmentID := uuid.New()
		fileName := fmt.Sprintf("%s%s", attachmentID.String(), ext)
		fullPath := filepath.Join(saveDir, fileName)

		if err := ctx.SaveUploadedFile(fileHeader, fullPath); err != nil {
			return models.OpportunityRemark{}, nil, fmt.Errorf("failed to save uploaded file %s: %v", fileHeader.Filename, err)
		}

		attachment := models.OpportunityRemarkAttachment{
			ID:                  attachmentID,
			OpportunityRemarkID: remarkID,
			FileName:            fileHeader.Filename,
			FilePath:            filepath.ToSlash(fullPath),
			FileType:            ext,
			FileSize:            fileHeader.Size,
			CreateDate:          now,
			CreateBy:            createBy,
			UpdateDate:          now,
			UpdateBy:            createBy,
		}

		if err := tx.Create(&attachment).Error; err != nil {
			return models.OpportunityRemark{}, nil, fmt.Errorf("failed to create opportunity remark attachment: %v", err)
		}

		attachments = append(attachments, attachment)
	}

	return row, attachments, nil
}
