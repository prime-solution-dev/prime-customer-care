package opportunityService

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

type UpdateOpportunityRemarkRequest struct {
	ID      uuid.UUID `json:"id"`
	Remark  *string   `json:"remark"`
	CsStaff *string   `json:"cs_staff"`
}

type UpdateOpportunityRemarkResponse struct {
	ResponseCode int    `json:"response_code"`
	Message      string `json:"message"`
}

func UpdateOpportunityRemarksRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req []UpdateOpportunityRemarkRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_prime_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.CloseGORM(gormx)

	return UpdateOpportunityRemarks(gormx, ctx, req)
}

func UpdateOpportunityRemarks(gormx *gorm.DB, ctx *gin.Context, req []UpdateOpportunityRemarkRequest) (*UpdateOpportunityRemarkResponse, error) {
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

	for _, item := range req {
		if item.ID == uuid.Nil {
			return nil, fmt.Errorf("id is required")
		}

		updateMap := map[string]interface{}{
			"update_date": now,
			"update_by":   userID,
		}

		if item.Remark != nil {
			updateMap["remark"] = *item.Remark
		}
		if item.CsStaff != nil {
			updateMap["cs_staff"] = *item.CsStaff
		}

		if err := gormx.Model(&models.OpportunityRemark{}).
			Where("id = ?", item.ID).
			Updates(updateMap).Error; err != nil {
			return nil, fmt.Errorf("failed to update opportunity remark %s: %w", item.ID, err)
		}
	}

	res := UpdateOpportunityRemarkResponse{
		ResponseCode: 200,
		Message:      "update success",
	}

	return &res, nil
}
