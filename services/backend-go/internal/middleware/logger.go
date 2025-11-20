package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Logger retourne un middleware de logging personnalisé
func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		entry := logger.WithFields(logrus.Fields{
			"status":   status,
			"method":   c.Request.Method,
			"path":     path,
			"query":    query,
			"duration": duration,
			"client":   c.ClientIP(),
		})

		if status >= 500 {
			entry.Error("Server error")
		} else if status >= 400 {
			entry.Warn("Client error")
		} else {
			entry.Info("Request processed")
		}
	}
}
