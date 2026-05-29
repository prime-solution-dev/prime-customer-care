package models

import "github.com/google/uuid"

type Brand struct {
	ID        uuid.UUID `json:"id"`
	BrandCode string    `json:"brand_code"`
	BrandName string    `json:"brand_name"`
}

func (Brand) TableName() string {
	return "brand"
}
