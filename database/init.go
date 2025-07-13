package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// DatabaseManager handles all database operations
type DatabaseManager struct {
	DB     *sql.DB
	DBPath string
	logger *zap.Logger
}

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// NewDatabaseManager creates a new database manager instance
func NewDatabaseManager(dbPath string, logger *zap.Logger) *DatabaseManager {
	return &DatabaseManager{
		DBPath: dbPath,
		logger: logger,
	}
}

// Connect establishes connection to the SQLite database
func (dm *DatabaseManager) Connect() error {
	db, err := sql.Open("sqlite3", dm.DBPath+"?_foreign_keys=on")
	if err != nil {
		dm.logger.Error("failed to open database:", zap.Error(err))
		return err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		dm.logger.Error("failed to ping database.", zap.Error(err))
		return err
	}

	dm.DB = db
	dm.logger.Info("Connected to database.", zap.String("Connected to database.", dm.DBPath))
	return nil
}

// Close closes the database connection
func (dm *DatabaseManager) Close() error {
	if dm.DB != nil {
		err := dm.DB.Close()
		if err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
		dm.logger.Info("Database connection closed")
	}
	return nil
}

// InitializeDatabase creates the database file and runs initial setup
func (dm *DatabaseManager) InitializeDatabase() error {
	if err := dm.Connect(); err != nil {
		return err
	}

	// Create migrations table if it doesn't exist
	if err := dm.createMigrationsTable(); err != nil {
		return err
	}

	// Run all migrations
	if err := dm.RunMigrations(); err != nil {
		return err
	}

	dm.logger.Info("Database initialized successfully")
	return nil
}

// createMigrationsTable creates the migrations tracking table
func (dm *DatabaseManager) createMigrationsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version INTEGER NOT NULL UNIQUE,
		name TEXT NOT NULL,
		executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := dm.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	return nil
}

// GetMigrations returns all available migrations
func GetMigrations() []Migration {
	return []Migration{
		{
			Version: 1,
			Name:    "create_users_table",
			SQL: `
			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT NOT NULL UNIQUE,
				email TEXT NOT NULL UNIQUE,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			
			CREATE INDEX idx_users_username ON users(username);
			CREATE INDEX idx_users_email ON users(email);
			`,
		},
	}
}

// RunMigrations executes all pending migrations
func (dm *DatabaseManager) RunMigrations() error {
	migrations := GetMigrations()

	for _, migration := range migrations {
		// Check if migration has already been executed
		var count int
		err := dm.DB.QueryRow("SELECT COUNT(*) FROM migrations WHERE version = ?", migration.Version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if count > 0 {
			dm.logger.Info("Migration already executed, skipping", zap.Int("migration version", migration.Version), zap.String("migration name", migration.Name))
			continue
		}

		// Execute migration
		log.Printf("Executing migration %d: %s", migration.Version, migration.Name)

		tx, err := dm.DB.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Execute the migration SQL
		_, err = tx.Exec(migration.SQL)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d: %w", migration.Version, err)
		}

		// Record the migration
		_, err = tx.Exec("INSERT INTO migrations (version, name) VALUES (?, ?)", migration.Version, migration.Name)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		dm.logger.Info("Migration %d (%s) executed successfully", zap.Int("migration version", migration.Version), zap.String("migration name", migration.Name))
	}

	return nil
}

// AddSampleData inserts some sample data for testing
func (dm *DatabaseManager) AddSampleData() error {
	log.Println("Adding sample data...")

	// Insert sample users
	userQueries := []string{
		"INSERT OR IGNORE INTO users (username, email) VALUES ('john_doe', 'john@example.com')",
		"INSERT OR IGNORE INTO users (username, email) VALUES ('jane_smith', 'jane@example.com')",
	}

	// Execute sample data queries
	allQueries := userQueries

	for _, query := range allQueries {
		_, err := dm.DB.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to insert sample data: %w", err)
		}
	}

	dm.logger.Info("Sample data added successfully")
	return nil
}

// GetTableInfo returns information about all tables
func (dm *DatabaseManager) GetTableInfo() error {
	rows, err := dm.DB.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()

	dm.logger.Info("Database Tables:")
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		dm.logger.Info("  - %s", zap.String("table name", tableName))
	}

	return nil
}

// Example usage and main function
// func main() {
// Create database manager
// dbManager := NewDatabaseManager("example.db")

// // Ensure cleanup
// defer func() {
// 	if err := dbManager.Close(); err != nil {
// 		log.Printf("Error closing database: %v", err)
// 	}
// }()

// // Initialize database
// if err := dbManager.InitializeDatabase(); err != nil {
// 	log.Fatalf("Failed to initialize database: %v", err)
// }

// // Add sample data
// if err := dbManager.AddSampleData(); err != nil {
// 	log.Printf("Warning: Failed to add sample data: %v", err)
// }

// // Display table information
// if err := dbManager.GetTableInfo(); err != nil {
// 	log.Printf("Warning: Failed to get table info: %v", err)
// }

// log.Println("Database setup completed successfully!")
// }

// Additional helper functions for extending the database

// ExecuteQuery executes a custom query and returns results
func (dm *DatabaseManager) ExecuteQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return dm.DB.Query(query, args...)
}

// ExecuteStatement executes a statement (INSERT, UPDATE, DELETE)
func (dm *DatabaseManager) ExecuteStatement(query string, args ...interface{}) (sql.Result, error) {
	return dm.DB.Exec(query, args...)
}

// BeginTransaction starts a new transaction
func (dm *DatabaseManager) BeginTransaction() (*sql.Tx, error) {
	return dm.DB.Begin()
}
