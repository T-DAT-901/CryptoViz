package controllers

import (
	"cryptoviz-backend/internal/dto"
	"cryptoviz-backend/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NewsController gère les endpoints pour les actualités crypto
type NewsController struct {
	repo   models.NewsRepository
	logger *logrus.Logger
}

// NewNewsController crée un nouveau NewsController
func NewNewsController(deps *Dependencies) *NewsController {
	return &NewsController{
		repo:   deps.NewsRepo,
		logger: deps.Logger,
	}
}

// GetAll récupère toutes les actualités
// @Summary Get all news
// @Description Récupère toutes les actualités crypto
// @Tags news
// @Param limit query int false "Limit" default(50)
// @Success 200 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/news [get]
func (ctrl *NewsController) GetAll(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 100 {
		limit = 50
	}

	data, err := ctrl.repo.GetAll(limit)
	if err != nil {
		ctrl.logger.Error("Erreur requête news: ", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Database error"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(data))
}

// GetBySymbol récupère les actualités par symbole
// @Summary Get news by symbol
// @Description Récupère les actualités pour un symbole spécifique
// @Tags news
// @Param symbol path string true "Symbol (ex: BTC/USDT)"
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/news/{symbol} [get]
func (ctrl *NewsController) GetBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 100 {
		limit = 20
	}

	data, err := ctrl.repo.GetBySymbol(symbol, limit)
	if err != nil {
		ctrl.logger.Error("Erreur requête news par symbole: ", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("Database error"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(data))
}
