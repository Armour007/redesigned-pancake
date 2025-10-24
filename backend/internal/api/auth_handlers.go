package api

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time" // Needed for created_at/updated_at

	"github.com/gin-gonic/gin"
	"github.com/google/uuid" // For generating user IDs

	// Adjust import paths as needed
	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/utils"
)

// Define User struct matching database table (or create in a models package)
type User struct {
	ID             uuid.UUID `db:"id"`
	FullName       string    `db:"full_name"`
	Email          string    `db:"email"`
	HashedPassword string    `db:"hashed_password"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// RegisterUser handles user registration requests
func RegisterUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// --- START NEW TRANSACTION LOGIC ---
	// We use a transaction to ensure all or nothing is created.
	tx, err := database.DB.Beginx() // Start a new transaction
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction"})
		return
	}
	// Defer a rollback in case anything fails
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in RegisterUser, rolling back transaction:", r)
			tx.Rollback()
		} else if err != nil {
			log.Println("Error in RegisterUser, rolling back transaction:", err)
			tx.Rollback()
		}
	}()

	// 1. Create the User
	newUser := User{
		ID:             uuid.New(),
		FullName:       req.FullName,
		Email:          req.Email,
		HashedPassword: hashedPassword,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	userQuery := `INSERT INTO users (id, full_name, email, hashed_password, created_at, updated_at)
				  VALUES (:id, :full_name, :email, :hashed_password, :created_at, :updated_at)`
	_, err = tx.NamedExec(userQuery, newUser)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Email address already registered"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		}
		return // This triggers the deferred rollback
	}

	// 2. Create the Organization
	newOrg := database.Organization{
		ID:        uuid.New(),
		Name:      req.FullName + "'s Organization", // Default org name
		OwnerID:   newUser.ID,                       // <-- Set the owner ID to the newly created user's ID
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// Update the query to include owner_id
	orgQuery := `INSERT INTO organizations (id, name, owner_id, created_at, updated_at)
             VALUES (:id, :name, :owner_id, :created_at, :updated_at)`
	_, err = tx.NamedExec(orgQuery, newOrg) // Use the updated struct and query
	// --- END CHANGE ---

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create default organization"})
		return // Triggers rollback
	}

	// 3. Link User to Organization as "owner"
	newMember := database.OrganizationMember{
		OrganizationID: newOrg.ID,
		UserID:         newUser.ID,
		Role:           "owner",
		JoinedAt:       time.Now(),
	}

	memberQuery := `INSERT INTO organization_members (organization_id, user_id, role, joined_at)
                VALUES (:organization_id, :user_id, :role, :joined_at)`
	_, err = tx.NamedExec(memberQuery, newMember)
	// --- END Check ---

	if err != nil {
		log.Printf("Error linking user to organization: %v\n", err) // Keep this log!
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link user to organization"})
		return // Triggers rollback
	}

	// 4. Commit the transaction
	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit registration transaction"})
		return
	}
	// --- END NEW TRANSACTION LOGIC ---

	// Respond with success
	c.JSON(http.StatusCreated, gin.H{
		"message":         "User registered successfully",
		"user_id":         newUser.ID,
		"email":           newUser.Email,
		"organization_id": newOrg.ID, // Also return the new Org ID!
	})
}

func LoginUser(c *gin.Context) {
	var req LoginRequest

	// Bind JSON request body and validate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Find the user by email in the database
	var user User
	query := `SELECT id, full_name, email, hashed_password, created_at, updated_at FROM users WHERE email=$1`
	err := database.DB.Get(&user, query, req.Email) // Use Get for single row

	if err != nil {
		if err == sql.ErrNoRows {
			// User not found
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		} else {
			// Other database error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		}
		return
	}

	// Check if the provided password matches the stored hash
	if !utils.CheckPasswordHash(req.Password, user.HashedPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Passwords match - Generate a JWT
	tokenString, err := utils.GenerateJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token: " + err.Error()})
		return
	}

	// Respond with the JWT token
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   tokenString,
		"user_id": user.ID,
	})
}
