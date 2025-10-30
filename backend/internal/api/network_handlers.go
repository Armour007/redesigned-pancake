package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// GET /v2/network/info — placeholder for network/tokenization readiness
func GetNetworkInfo(c *gin.Context) {
	enabled := os.Getenv("AURA_NETWORK_ENABLE") == "1"
	out := gin.H{"enabled": enabled}
	if !enabled {
		out["note"] = "network features disabled; pending governance/legal readiness"
	} else {
		out["utility_token"] = map[string]any{
			"symbol":          os.Getenv("AURA_NETWORK_UTILITY_TOKEN_SYMBOL"),
			"staking_enabled": os.Getenv("AURA_NETWORK_STAKING_ENABLE") == "1",
		}
	}
	c.JSON(http.StatusOK, out)
}
