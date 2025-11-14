package controllers

import (
	"cryptoviz-backend/internal/dto"
	"cryptoviz-backend/models"
	"net/http"
	"strconv"

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
// @Description Récupère les indicateurs techniques d'un type spécifique
// @Tags indicators
// @Param symbol path string true "Symbol (ex: BTC/USDT)"
// @Param type path string true "Indicator type (ex: rsi, macd, bollinger)"
// @Param interval query string false "Interval" default(1m)
// @Param limit query int false "Limit" default(100)
// @Success 200 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/indicators/{symbol}/{type} [get]
func (ctrl *IndicatorController) GetByType(c *gin.Context) {
	symbol := c.Param("symbol")
	indicatorType := c.Param("type")
	interval := c.DefaultQuery("interval", "1m")
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 1000 {
		limit = 100
	}

	data, err := ctrl.repo.GetBySymbolAndType(symbol, indicatorType, interval, limit)
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
// @Param symbol path string true "Symbol (ex: BTC/USDT)"
// @Param interval query string false "Interval" default(1m)
// @Success 200 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/indicators/{symbol} [get]
func (ctrl *IndicatorController) GetAll(c *gin.Context) {
	symbol := c.Param("symbol")
	interval := c.DefaultQuery("interval", "1m")

	data, err := ctrl.repo.GetAllBySymbol(symbol, interval)
	if err != nil {
		ctrl.logger.Error("Erreur requête tous indicateurs: ", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Database error"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(data))
}
