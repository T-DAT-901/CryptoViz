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
// @Description Récupère les données OHLCV pour un symbole avec support des plages de temps
// @Tags candles
// @Param symbol query string true "Symbol (ex: BTC/USDT)"
// @Param interval query string false "Interval (ex: 1m, 5m, 1h)" default(1m)
// @Param limit query int false "Limit (max 5000)" default(100)
// @Param start_time query string false "Start time (RFC3339 format)"
// @Param end_time query string false "End time (RFC3339 format)"
// @Success 200 {object} dto.APIResponse
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/crypto/data [get]
func (ctrl *CandleController) GetCandleData(c *gin.Context) {
	// Symbol is validated and normalized by middleware, stored in context
	symbol, _ := c.Get("symbol")
	symbolStr := symbol.(string)
	interval := c.DefaultQuery("interval", "1m")
	limitStr := c.DefaultQuery("limit", "100")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	// Parse limit with significantly increased maximum to support full historical data
	// For 40 days of 1m candles: ~57,600 candles
	// For 5 years of 1m candles: ~2,628,000 candles
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
		allData, err := ctrl.repo.GetAllByTimeRange(symbolStr, interval, startTime, endTime)
		if err != nil {
			ctrl.logger.Error("Erreur requête crypto data (time range): ", err)
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Database error"))
			return
		}

		// Convert AllCandle to Candle for response
		data := make([]models.Candle, len(allData))
		for i, ac := range allData {
			data[i] = ac.Candle
		}

		c.JSON(http.StatusOK, dto.SuccessResponse(data))
		return
	}

	// Default behavior: use limit-based query on unified view for better historical data access
	allData, err := ctrl.repo.GetAllBySymbol(symbolStr, interval, limit)
	if err != nil {
		ctrl.logger.Error("Erreur requête crypto data: ", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Database error"))
		return
	}

	// Convert AllCandle to Candle for response
	data := make([]models.Candle, len(allData))
	for i, ac := range allData {
		data[i] = ac.Candle
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

// GetAvailableSymbols récupère la liste des symboles disponibles
// @Summary Get available symbols
// @Description Récupère la liste des symboles ayant des données récentes (dernière heure)
// @Tags candles
// @Success 200 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/crypto/symbols [get]
func (ctrl *CandleController) GetAvailableSymbols(c *gin.Context) {
	symbols, err := ctrl.repo.GetDistinctSymbols()
	if err != nil {
		ctrl.logger.Error("Erreur requête symbols: ", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Database error"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"symbols": symbols}))
}
