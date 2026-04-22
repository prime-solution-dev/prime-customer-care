package opportunityService

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"prime-customer-care/internal/db"
	"prime-customer-care/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func DownloadOpportunityRemarkAttachmentRest(ctx *gin.Context, _ string) (interface{}, error) {
	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_prime_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return DownloadOpportunityRemarkAttachment(ctx, gormx)
}

func DownloadOpportunityRemarkAttachment(ctx *gin.Context, gormx *gorm.DB) (interface{}, error) {
	if gormx == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	idStr := strings.TrimSpace(ctx.Query("id"))
	if idStr == "" {
		return nil, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %v", err)
	}

	var row models.OpportunityRemarkAttachment
	if err := gormx.Where("id = ?", id).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("attachment not found")
		}
		return nil, fmt.Errorf("failed to get attachment: %v", err)
	}

	fullPath := strings.TrimSpace(row.FilePath)
	if fullPath == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	if _, err := os.Stat(fullPath); err != nil {
		return nil, fmt.Errorf("file not found on disk")
	}

	fileName := strings.TrimSpace(row.FileName)
	if fileName == "" {
		fileName = filepath.Base(fullPath)
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fullPath)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	ctx.Header("Content-Type", contentType)
	ctx.File(fullPath)

	return nil, nil
}
