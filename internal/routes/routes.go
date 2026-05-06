package routes

import (
	brandService "prime-customer-care/internal/services/brand-service"
	opportunityService "prime-customer-care/internal/services/opportunity-service"
	ticketService "prime-customer-care/internal/services/ticket-service"
	ticketTypeService "prime-customer-care/internal/services/ticket-type-service"
	"prime-customer-care/internal/utils"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(ctx *gin.Engine) {

	// Brand
	brandRoutes := ctx.Group("/brand")

	brandRoutes.POST("/get-brands", func(c *gin.Context) {
		utils.ProcessRequest(c, brandService.GetBrandsRest)
	})

	// Ticket Type
	ticketTypeRoutes := ctx.Group("/ticket-type")

	ticketTypeRoutes.POST("/get-ticket-types", func(c *gin.Context) {
		utils.ProcessRequest(c, ticketTypeService.GetTicketTypesRest)
	})

	//Ticket
	ticketRoutes := ctx.Group("/ticket")

	ticketRoutes.POST("/get-tickets", func(c *gin.Context) {
		utils.ProcessRequest(c, ticketService.GetTicketsRest)
	})
	ticketRoutes.POST("/get-ticket-overview", func(c *gin.Context) {
		utils.ProcessRequest(c, ticketService.GetTicketOverviewRest)
	})
	ticketRoutes.POST("/create-tickets", func(c *gin.Context) {
		utils.ProcessRequest(c, ticketService.CreateTicketsRest)
	})
	ticketRoutes.POST("/update-tickets", func(c *gin.Context) {
		utils.ProcessRequest(c, ticketService.UpdateTicketsRest)
	})

	// Opportunity
	opportunityRoutes := ctx.Group("/opportunity")

	opportunityRoutes.POST("/get-opportunities", func(c *gin.Context) {
		utils.ProcessRequest(c, opportunityService.GetOpportunitiesRest)
	})
	opportunityRoutes.POST("/create-opportunities", func(c *gin.Context) {
		utils.ProcessRequest(c, opportunityService.CreateOpportunitiesRest)
	})
	opportunityRoutes.POST("/create-opportunity-tickets", func(c *gin.Context) {
		utils.ProcessRequest(c, opportunityService.CreateOpportunityTicketsRest)
	})
	opportunityRoutes.POST("/create-opportunity-remarks", func(c *gin.Context) {
		utils.ProcessRequest(c, opportunityService.CreateOpportunityRemarkRest)
	})
	opportunityRoutes.POST("/update-opportunities", func(c *gin.Context) {
		utils.ProcessRequest(c, opportunityService.UpdateOpportunitiesRest)
	})
	opportunityRoutes.POST("/update-opportunity-remarks", func(c *gin.Context) {
		utils.ProcessRequest(c, opportunityService.UpdateOpportunityRemarksRest)
	})

}
