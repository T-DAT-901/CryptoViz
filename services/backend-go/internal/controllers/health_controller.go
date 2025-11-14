package controllers

import (
	"context"
	"cryptoviz-backend/database"
	"cryptoviz-backend/internal/dto"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// HealthController gère les endpoints de santé
type HealthController struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewHealthController crée un nouveau HealthController
func NewHealthController(deps *Dependencies) *HealthController {
	return &HealthController{
		db:    deps.DB,
		redis: deps.Redis,
	}
}

// HealthCheck vérifie que le service est en vie
// @Summary Health check
// @Description Vérifie que le service est en vie
// @Tags health
// @Success 200 {object} dto.APIResponse
// @Router /health [get]
func (ctrl *HealthController) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, dto.SuccessMessageResponse("Service healthy", nil))
}

// ReadinessCheck vérifie que le service est prêt (DB et Redis connectés)
// @Summary Readiness check
// @Description Vérifie que le service est prêt avec connexions DB et Redis
// @Tags health
// @Success 200 {object} dto.APIResponse
// @Failure 503 {object} dto.APIResponse
// @Router /ready [get]
func (ctrl *HealthController) ReadinessCheck(c *gin.Context) {
	// Vérifier la connexion GORM
	if err := database.Health(); err != nil {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse("Database not ready"))
		return
	}

	// Vérifier Redis
	ctx := context.Background()
	if err := ctrl.redis.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse("Redis not ready"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessMessageResponse("Service ready", nil))
}
