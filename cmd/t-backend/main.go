package main

import (
	"fmt"
	"os"

	db "github.com/chrisp986/trader-backend/database"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// // HealthResponse represents the health check response structure
// type HttpResponse struct {
// 	HttpStatusCode int       `json:"http_status_code"`
// 	Status         string    `json:"status"`
// 	Timestamp      time.Time `json:"timestamp"`
// 	Version        string    `json:"version"`
// 	Uptime         string    `json:"uptime"`
// }

type application struct {
	logger *zap.Logger
	user   db.UserModelInterface
}

type config struct {
	port     string
	dbPath   string
	logLevel string
}

// newLogger creates a new zap logger with structured JSON output
func newLogger(logLevel string) *zap.Logger {

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		// Create a temporary logger to log the warning
		tempLogger, _ := zap.NewProduction()
		tempLogger.Warn("Invalid log level, defaulting to INFO", zap.String("provided_level", logLevel), zap.Error(err))
		tempLogger.Sync()
		level = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	return logger
}

func getConfig() config {
	// Get log level from environment variable or default to INFO
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	cfg := config{port: port, dbPath: "trader_backend.db", logLevel: logLevel}
	return cfg
}

func main() {

	cfg := getConfig()

	logger := newLogger(cfg.logLevel)

	// Create database manager
	dbManager := db.NewDatabaseManager(cfg.dbPath, logger)
	app := &application{user: &db.UserModel{DB: dbManager.DB, Logger: logger}}

	// Ensure cleanup
	defer func() {
		if err := dbManager.Close(); err != nil {
			logger.Info("Error closing database:", zap.Error(err))
		}
	}()

	// Initialize database
	if err := dbManager.InitializeDatabase(); err != nil {
		logger.Fatal("Failed to initialize database:", zap.Error(err))
	}

	// // Add sample data
	// if err := dbManager.AddSampleData(); err != nil {
	// 	log.Printf("Warning: Failed to add sample data: %v", err)
	// }

	// Display table information
	if err := dbManager.GetTableInfo(); err != nil {
		logger.Info("Warning: Failed to get table info:", zap.Error(err))
	}

	logger.Info("Database setup completed successfully!")

	server := NewServer()

	// Ensure logger is properly closed on exit
	defer logger.Sync()

	addr := ":" + cfg.port

	fmt.Println("Starting Trader backend with address:", addr)
	logger.Info("Application starting",
		zap.String("port", cfg.port),
	)

	if err := server.Start(addr); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
