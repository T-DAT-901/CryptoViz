package controllers

import (
	"cryptoviz-backend/internal/dto"
	"cryptoviz-backend/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CandleController gère les endpoints pour les candles OHLCV
type CandleController struct {
	repo   models.CandleRepository
	logger *logrus.Logger
}

// NewCandleController crée un nouveau CandleController
func NewCandleController(deps *Dependencies) *CandleController {
	return &CandleController{
		repo:   deps.CandleRepo,
		logger: deps.Logger,
	}
}

// GetCandleData récupère les données OHLCV pour un symbole
// @Summary Get candle data
// @Description Récupère les données OHLCV pour un symbole
// @Tags candles
// @Param symbol query string true "Symbol (ex: BTC/USDT)"
// @Param interval query string false "Interval (ex: 1m, 5m, 1h)" default(1m)
// @Param limit query int false "Limit" default(100)
// @Success 200 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/crypto/data [get]
func (ctrl *CandleController) GetCandleData(c *gin.Context) {
	// Symbol is validated and normalized by middleware, stored in context
	symbol, _ := c.Get("symbol")
	symbolStr := symbol.(string)
	interval := c.DefaultQuery("interval", "1m")
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 1000 {
		limit = 100
	}

	data, err := ctrl.repo.GetBySymbol(symbolStr, interval, limit)
	if err != nil {
		ctrl.logger.Error("Erreur requête crypto data: ", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Database error"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(data))
}

// GetLatestPrice récupère le dernier prix pour un symbole
// @Summary Get latest price
// @Description Récupère le dernier prix pour un symbole
// @Tags candles
// @Param symbol query string true "Symbol (ex: BTC/USDT)"
// @Param exchange query string false "Exchange" default(BINANCE)
// @Success 200 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Router /api/v1/crypto/latest [get]
func (ctrl *CandleController) GetLatestPrice(c *gin.Context) {
	// Symbol is validated and normalized by middleware, stored in context
	symbol, _ := c.Get("symbol")
	symbolStr := symbol.(string)
	exchange := c.DefaultQuery("exchange", "BINANCE")

	data, err := ctrl.repo.GetLatest(symbolStr, exchange)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("No data found"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(data))
}

// GetStats récupère les statistiques pour un symbole
// @Summary Get crypto statistics
// @Description Récupère les statistiques (min, max, avg, etc.) pour un symbole
// @Tags candles
// @Param symbol query string true "Symbol (ex: BTC/USDT)"
// @Param interval query string false "Interval" default(1m)
// @Success 200 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Router /api/v1/stats [get]
func (ctrl *CandleController) GetStats(c *gin.Context) {
	// Symbol is validated and normalized by middleware, stored in context
	symbol, _ := c.Get("symbol")
	symbolStr := symbol.(string)
	interval := c.DefaultQuery("interval", "1m")

	stats, err := ctrl.repo.GetStats(symbolStr, interval)
	if err != nil {
		ctrl.logger.Error("Erreur requête stats: ", err)
		c.JSON(http.StatusNotFound, dto.ErrorResponse("No stats found"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(stats))
}
