package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SupportedIntervals defines the list of valid timeframe intervals
var SupportedIntervals = []string{"1m", "5m", "15m", "1h", "1d"}

// ValidateInterval validates the interval query parameter
// Returns 400 Bad Request if interval is invalid
// Usage: router.GET("/data", middleware.ValidateInterval(), handler)
func ValidateInterval() gin.HandlerFunc {
	return func(c *gin.Context) {
		interval := c.DefaultQuery("interval", "1m")

		// Check if interval is valid
		valid := false
		for _, supported := range SupportedIntervals {
			if interval == supported {
				valid = true
				break
			}
		}

		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid interval. Supported intervals: 1m, 5m, 15m, 1h, 1d",
				"example": "/api/v1/crypto/data?symbol=BTC/USDT&interval=1m",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
