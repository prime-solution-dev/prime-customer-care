package opportunityService

import (
	"fmt"
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

func CreateOpportunityRemarkRest(ctx *gin.Context) (interface{}, error) {
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

	now := time.Now()
	remarkID := uuid.New()

	basePath := strings.TrimSpace(os.Getenv("path_attachment"))
	if basePath == "" {
		basePath = "./attachment"
	}

	row := models.OpportunityRemark{
		ID:            remarkID,
		OpportunityID: opportunityID,
		Remark:        remark,
		CsStaff:       csStaff,
		CreateDate:    now,
		CreateBy:      createBy,
		UpdateDate:    now,
		UpdateBy:      createBy,
	}

	attachments := []models.OpportunityRemarkAttachment{}

	err = gormx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("failed to create opportunity remark: %v", err)
		}

		form, err := ctx.MultipartForm()
		if err != nil {
			return nil
		}

		fileHeaders := form.File["files"]
		if len(fileHeaders) == 0 {
			fileHeaders = form.File["file"]
		}
		if len(fileHeaders) == 0 {
			return nil
		}

		saveDir := filepath.Join(basePath, "opportunity_remark", remarkID.String())
		if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
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

		for _, fileHeader := range fileHeaders {
			ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
			if !allowedExt[ext] {
				return fmt.Errorf("file type not allowed: %s", ext)
			}

			if fileHeader.Size > 10*1024*1024 {
				return fmt.Errorf("file size exceeds 10MB: %s", fileHeader.Filename)
			}

			attachmentID := uuid.New()
			fileName := fmt.Sprintf("%s%s", attachmentID.String(), ext)
			fullPath := filepath.Join(saveDir, fileName)

			if err := ctx.SaveUploadedFile(fileHeader, fullPath); err != nil {
				return fmt.Errorf("failed to save uploaded file %s: %v", fileHeader.Filename, err)
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
				return fmt.Errorf("failed to create opportunity remark attachment: %v", err)
			}

			attachments = append(attachments, attachment)
		}

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
