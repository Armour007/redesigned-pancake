package api

import (
	"net/http"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

type MeResponse struct {
	ID        string    `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetMe returns the current authenticated user's basic profile
func GetMe(c *gin.Context) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var row struct {
		ID        string    `db:"id"`
		FullName  string    `db:"full_name"`
		Email     string    `db:"email"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := database.DB.Get(&row, `SELECT id, full_name, email, created_at, updated_at FROM users WHERE id=$1`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load profile"})
		return
	}
	c.JSON(http.StatusOK, MeResponse{
		ID:        row.ID,
		FullName:  row.FullName,
		Email:     row.Email,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	})
}

type UpdateMeRequest struct {
	FullName *string `json:"full_name"`
}

// UpdateMe updates simple profile attributes like full_name
func UpdateMe(c *gin.Context) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if req.FullName == nil || *req.FullName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "full_name is required"})
		return
	}
	_, err := database.DB.Exec(`UPDATE users SET full_name=$1, updated_at=NOW() WHERE id=$2`, *req.FullName, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}
	GetMe(c)
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UpdatePassword changes the current user's password after verifying current password
func UpdatePassword(c *gin.Context) {
	uid, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if ok, why := utils.ValidatePasswordPolicy(req.NewPassword); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": why})
		return
	}
	var storedHash string
	err := database.DB.Get(&storedHash, `SELECT hashed_password FROM users WHERE id=$1`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}
	if !utils.CheckPasswordHash(req.CurrentPassword, storedHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}
	newHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	_, err = database.DB.Exec(`UPDATE users SET hashed_password=$1, updated_at=NOW() WHERE id=$2`, newHash, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password updated"})
}
