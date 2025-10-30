package api

import (
	"net/http"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
)

type peerRow struct {
	ID  string `db:"id" json:"id"`
	URL string `db:"url" json:"url"`
}

// GET /v2/federation/peers
func ListFederationPeers(c *gin.Context) {
	rows := []peerRow{}
	if err := database.DB.Select(&rows, `SELECT id::text, url FROM federation_peers ORDER BY added_at DESC`); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "count": len(rows)})
}

// POST /v2/federation/peers { url }
func AddFederationPeer(c *gin.Context) {
	var body struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url required"})
		return
	}
	if _, err := database.DB.Exec(`INSERT INTO federation_peers(url) VALUES ($1)`, body.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusCreated)
}

// DELETE /v2/federation/peers/:peerId
func DeleteFederationPeer(c *gin.Context) {
	id := c.Param("peerId")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "peerId required"})
		return
	}
	if _, err := database.DB.Exec(`DELETE FROM federation_peers WHERE id::text=$1`, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
