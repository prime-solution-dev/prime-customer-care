package ticketTypeService

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"prime-customer-care/internal/db"
	"prime-customer-care/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetTicketTypesRequest struct {
	ID        []string `json:"id"`
	TypeCode  []string `json:"type_code"`
	TypeName  []string `json:"type_name"`
	Page      int      `json:"page"`
	PageSize  int      `json:"page_size"`
}

type GetTicketTypesResponse struct {
	ResponseCode int                 `json:"response_code"`
	Message      string              `json:"message"`
	Page         int                 `json:"page"`
	TotalPages   int                 `json:"total_pages"`
	Total        int64               `json:"total"`
	TicketTypes  []models.TicketType `json:"ticket_types"`
}

func GetTicketTypesRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req GetTicketTypesRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return GetTicketTypes(gormx, req)
}

func GetTicketTypes(gormx *gorm.DB, request GetTicketTypesRequest) (*GetTicketTypesResponse, error) {
	if gormx == nil {
		return nil, errors.New("database connection is nil")
	}

	usePagination := request.PageSize > 0

	res := GetTicketTypesResponse{
		ResponseCode: 200,
		Message:      "success",
		Page:         request.Page,
		TicketTypes:  []models.TicketType{},
	}

	query := gormx.Model(&models.TicketType{})

	if len(request.ID) > 0 {
		query = query.Where("id IN ?", request.ID)
	}

	if len(request.TypeCode) > 0 {
		query = query.Where("type_code IN ?", request.TypeCode)
	}

	if len(request.TypeName) > 0 {
		query = query.Where("type_name IN ?", request.TypeName)
	}

	var total int64
	if err := query.Debug().Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count ticket types: %v", err)
	}

	fetchQuery := query.Order("create_dtm DESC").Order("id DESC")
	totalPages := 0

	if usePagination {
		page := request.Page
		if page <= 0 {
			page = 1
		}

		pageSize := request.PageSize
		if total > 0 {
			totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
		}

		res.Page = page
		res.Total = total
		res.TotalPages = totalPages

		if total == 0 || page > totalPages {
			return &res, nil
		}

		offset := (page - 1) * pageSize
		fetchQuery = fetchQuery.Limit(pageSize).Offset(offset)
	} else if total > 0 {
		res.Page = 1
		totalPages = 1
	}

	var ticketTypes []models.TicketType
	if err := fetchQuery.Find(&ticketTypes).Error; err != nil {
		return nil, fmt.Errorf("failed to get ticket types: %v", err)
	}

	res.Total = total
	res.TotalPages = totalPages
	res.TicketTypes = ticketTypes

	return &res, nil
}
