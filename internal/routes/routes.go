package routes

import (
	ticketService "prime-customer-care/internal/services/ticket-service"
	"prime-customer-care/internal/utils"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(ctx *gin.Engine) {

	//Ticket
	ticketRoutes := ctx.Group("/ticket")

	ticketRoutes.POST("/get-tickets", func(c *gin.Context) {
		utils.ProcessRequest(c, ticketService.GetTicketsRest)
	})
	ticketRoutes.POST("/create-tickets", func(c *gin.Context) {
		utils.ProcessRequest(c, ticketService.CreateTicketsRest)
	})
	ticketRoutes.POST("/update-tickets", func(c *gin.Context) {
		utils.ProcessRequest(c, ticketService.UpdateTicketsRest)
	})

}
