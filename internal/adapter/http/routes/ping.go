package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// @Summary Health check endpoint
// @Description Returns a pong message to verify the API is running
// @Tags Health Check
// @Security Bearer
// @Produce json
// @Success 200 {object} map[string]interface{} "Returns pong message"
// @Router /ping [get]
func addPingRoutes(rg *gin.RouterGroup) {
	ping := rg.Group(PathHealthCheck)
	{
		ping.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}
}
