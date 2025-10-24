package database

import (
	"fmt"
	"log"
	"os" // Needed to read environment variables

	_ "github.com/jackc/pgx/v5/stdlib" // Postgres driver - the underscore means we need its side effects (registration) but won't call it directly
	"github.com/jmoiron/sqlx"          // Provides helpful extensions for database/sql
	"github.com/joho/godotenv"         // Used to load .env files
)

// DB holds the database connection pool (exported so other packages can use it)
var DB *sqlx.DB

// Connect initializes the database connection using variables from the .env file
func Connect() {
	// --- Load .env file ---
	// This will look for a file named ".env" in the directory
	// from which the 'go run' command is executed.
	// Since we will run from the project root ('aura-project'), it will find 'aura-project/.env'
	err := godotenv.Load(".env")
	if err != nil {
		// Log a warning but don't stop the program, as env vars might be set in the system
		log.Println("Warning: Could not load .env file:", err)
	}

	// --- Read Database Configuration from Environment Variables ---
	// Provide default values in case they are not set in the .env file or system environment
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost" // Default host
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432" // Default PostgreSQL port
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "aura_user" // Default user from docker-compose.yml
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		// If the password is truly missing, we cannot continue. Log Fatal to stop.
		log.Fatal("FATAL: DB_PASSWORD environment variable is not set or .env file not loaded correctly")
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "aura_db" // Default database name from docker-compose.yml
	}
	dbSSLMode := os.Getenv("DB_SSLMODE")
	if dbSSLMode == "" {
		dbSSLMode = "disable" // Default for local development
	}

	// --- Construct the Connection String ---
	// Example: "host=localhost port=5432 user=aura_user password=your_password dbname=aura_db sslmode=disable"
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// --- Open Database Connection ---
	// sqlx.Connect combines opening the connection and pinging it
	db, err := sqlx.Connect("pgx", connStr)
	if err != nil {
		// If we can't connect, log Fatal to stop the application
		log.Fatalf("FATAL: Unable to connect to database: %v\n", err)
	}

	// sqlx.Connect already pings, but an explicit Ping can be added if needed for extra verification
	// err = db.Ping()
	// if err != nil {
	//  log.Fatalf("FATAL: Unable to ping database after connect: %v\n", err)
	// }

	// --- Store Connection Globally ---
	// Assign the successful connection pool to the global DB variable
	DB = db
	log.Println("Successfully connected to the database!") // Success message 🎉
}

// Close function (optional but good practice for graceful shutdown)
// func Close() {
// 	if DB != nil {
// 		err := DB.Close()
// 		if err != nil {
// 			log.Printf("Error closing database connection: %v\n", err)
// 		} else {
// 			log.Println("Database connection closed.")
// 		}
// 	}
// }
