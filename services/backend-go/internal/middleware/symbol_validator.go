package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// SymbolValidator validates cryptocurrency trading pair symbols in query parameters
// Expects format like "BTC/USDT", "ETH/BTC", etc.
// Query parameters naturally support slashes without encoding

var (
	// symbolPattern matches trading pairs like BTC/USDT, ETH/BTC, etc.
	// Format: BASE/QUOTE where both parts are 2-10 uppercase letters/digits/hyphens
	symbolPattern = regexp.MustCompile(`^[A-Z0-9-]{2,10}/[A-Z0-9-]{2,10}$`)
)

// ValidateSymbolQuery validates the ?symbol= query parameter
// Usage: router.GET("/crypto/data", middleware.ValidateSymbolQuery(), handler)
func ValidateSymbolQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := c.Query("symbol")

		// Symbol is required
		if symbol == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Missing required query parameter: symbol",
				"example": "/api/v1/crypto/data?symbol=BTC/USDT",
			})
			c.Abort()
			return
		}

		// Special case: allow "*" for wildcard subscriptions
		if symbol == "*" {
			c.Next()
			return
		}

		// Validate symbol format
		if !symbolPattern.MatchString(symbol) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid symbol format. Expected format: BASE/QUOTE (e.g., BTC/USDT)",
				"example": "/api/v1/crypto/data?symbol=BTC/USDT",
			})
			c.Abort()
			return
		}

		// Normalize symbol (ensure uppercase) and set in context
		normalizedSymbol := strings.ToUpper(symbol)
		c.Set("symbol", normalizedSymbol)

		c.Next()
	}
}

// ValidateSymbolHyphen validates symbol_hyphen parameter (for alternative routes)
// Accepts hyphen format like "BTC-USDT" and converts to "BTC/USDT" for database queries
func ValidateSymbolHyphen() gin.HandlerFunc {
	return func(c *gin.Context) {
		symbolHyphen := c.Param("symbol_hyphen")

		if symbolHyphen == "" {
			c.Next()
			return
		}

		// Convert hyphen to slash for database compatibility
		symbol := strings.ReplaceAll(symbolHyphen, "-", "/")

		// Validate converted format
		if !symbolPattern.MatchString(symbol) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid symbol format. Expected format: BASE-QUOTE (e.g., BTC-USDT)",
			})
			c.Abort()
			return
		}

		// Store converted symbol in context for handler access
		c.Set("symbol", strings.ToUpper(symbol))
		c.Next()
	}
}

// replaceParam replaces a parameter value in gin.Params
func replaceParam(params gin.Params, key, value string) gin.Params {
	newParams := make(gin.Params, len(params))
	for i, p := range params {
		if p.Key == key {
			newParams[i] = gin.Param{Key: key, Value: value}
		} else {
			newParams[i] = p
		}
	}
	return newParams
}
