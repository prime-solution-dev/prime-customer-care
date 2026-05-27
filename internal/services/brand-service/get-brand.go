package brandService

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

type GetBrandsRequest struct {
	ID        []string `json:"id"`
	BrandCode []string `json:"brand_code"`
	BrandName []string `json:"brand_name"`
	Page      int      `json:"page"`
	PageSize  int      `json:"page_size"`
}

type GetBrandsResponse struct {
	ResponseCode int            `json:"response_code"`
	Message      string         `json:"message"`
	Page         int            `json:"page"`
	TotalPages   int            `json:"total_pages"`
	Total        int64          `json:"total"`
	Brands       []models.Brand `json:"brands"`
}

func GetBrandsRest(ctx *gin.Context, jsonPayload string) (interface{}, error) {
	var req GetBrandsRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	gormx, err := db.ConnectGORM(os.Getenv("database_sqlx_url_customer_care"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.CloseGORM(gormx)

	return GetBrands(gormx, req)
}

func GetBrands(gormx *gorm.DB, request GetBrandsRequest) (*GetBrandsResponse, error) {
	if gormx == nil {
		return nil, errors.New("database connection is nil")
	}

	usePagination := request.PageSize > 0

	res := GetBrandsResponse{
		ResponseCode: 200,
		Message:      "success",
		Page:         request.Page,
		Brands:       []models.Brand{},
	}

	query := gormx.Model(&models.Brand{})

	if len(request.ID) > 0 {
		query = query.Where("id IN ?", request.ID)
	}

	if len(request.BrandCode) > 0 {
		query = query.Where("brand_code IN ?", request.BrandCode)
	}

	if len(request.BrandName) > 0 {
		query = query.Where("brand_name IN ?", request.BrandName)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count brands: %v", err)
	}

	fetchQuery := query.Order("brand_code ASC")
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

	var brands []models.Brand
	if err := fetchQuery.Find(&brands).Error; err != nil {
		return nil, fmt.Errorf("failed to get brands: %v", err)
	}

	res.Total = total
	res.TotalPages = totalPages
	res.Brands = brands

	return &res, nil
}
