package controllers

import (
	"cryptoviz-backend/internal/dto"
	"cryptoviz-backend/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// IndicatorController gère les endpoints pour les indicateurs techniques
type IndicatorController struct {
	repo   models.IndicatorRepository
	logger *logrus.Logger
}

// NewIndicatorController crée un nouveau IndicatorController
func NewIndicatorController(deps *Dependencies) *IndicatorController {
	return &IndicatorController{
		repo:   deps.IndicatorRepo,
		logger: deps.Logger,
	}
}

// GetByType récupère les indicateurs par type
// @Summary Get indicators by type
// @Description Récupère les indicateurs techniques d'un type spécifique avec support des plages de temps
// @Tags indicators
// @Param symbol query string true "Symbol (ex: BTC/USDT)"
// @Param type path string true "Indicator type (ex: rsi, macd, bollinger)"
// @Param interval query string false "Interval" default(1m)
// @Param limit query int false "Limit (max 5000)" default(100)
// @Param start_time query string false "Start time (RFC3339 format)"
// @Param end_time query string false "End time (RFC3339 format)"
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/indicators/{type} [get]
func (ctrl *IndicatorController) GetByType(c *gin.Context) {
	// Symbol is validated and normalized by middleware, stored in context
	symbol, _ := c.Get("symbol")
	symbolStr := symbol.(string)
	indicatorType := c.Param("type")
	interval := c.DefaultQuery("interval", "1m")
	limitStr := c.DefaultQuery("limit", "100")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	// Parse limit with significantly increased maximum to support full historical data
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 100000 {
		limit = 100
	}

	// If time range is provided, use time-based query on unified view (hot + cold storage)
	if startTimeStr != "" && endTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("Invalid start_time format. Use RFC3339 format (e.g., 2024-01-01T00:00:00Z)"))
			return
		}

		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("Invalid end_time format. Use RFC3339 format (e.g., 2024-01-01T00:00:00Z)"))
			return
		}

		// Use unified view to query both hot and cold storage
		allData, err := ctrl.repo.GetAllByTimeRangeUnified(symbolStr, indicatorType, interval, startTime, endTime)
		if err != nil {
			ctrl.logger.Error("Erreur requête indicateurs (time range): ", err)
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Database error"))
			return
		}

		// Convert AllIndicator to Indicator for response
		data := make([]models.Indicator, len(allData))
		for i, ai := range allData {
			data[i] = ai.Indicator
		}

		c.JSON(http.StatusOK, dto.SuccessResponse(data))
		return
	}

	// Default behavior: use limit-based query (hot storage)
	// Note: For better historical access, we could switch to unified view here too
	data, err := ctrl.repo.GetBySymbolAndType(symbolStr, indicatorType, interval, limit)
	if err != nil {
		ctrl.logger.Error("Erreur requête indicateurs: ", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Database error"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(data))
}

// GetAll récupère tous les indicateurs pour un symbole
// @Summary Get all indicators
// @Description Récupère tous les indicateurs techniques pour un symbole
// @Tags indicators
// @Param symbol query string true "Symbol (ex: BTC/USDT)"
// @Param interval query string false "Interval" default(1m)
// @Success 200 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/indicators [get]
func (ctrl *IndicatorController) GetAll(c *gin.Context) {
	// Symbol is validated and normalized by middleware, stored in context
	symbol, _ := c.Get("symbol")
	symbolStr := symbol.(string)
	interval := c.DefaultQuery("interval", "1m")

	data, err := ctrl.repo.GetAllBySymbol(symbolStr, interval)
	if err != nil {
		ctrl.logger.Error("Erreur requête tous indicateurs: ", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Database error"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(data))
}
