package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Anshul1310/tf-register/config"
	"github.com/Anshul1310/tf-register/internals/database"
	"github.com/Anshul1310/tf-register/internals/handlers"
	"github.com/Anshul1310/tf-register/internals/repository"
	"github.com/Anshul1310/tf-register/internals/routes"
	"github.com/Anshul1310/tf-register/internals/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// 1. Load application configuration
	applicationConfig := config.LoadConfig()

	// 2. Initialize PostgreSQL connection pool
	databaseConnectionPool, databaseError := database.InitializeDatabasePool(applicationConfig.DatabaseURL)
	if databaseError != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", databaseError)
	}
	defer database.CloseDatabasePool(databaseConnectionPool)

	// 3. Initialize Repositories
	userRepository := repository.NewUserRepository(databaseConnectionPool)
	teamRepository := repository.NewTeamRepository(databaseConnectionPool)

	// 4. Initialize Services
	userService := service.NewUserService(userRepository, teamRepository, applicationConfig)
	if err := userService.EnsureMasterUser(context.Background()); err != nil {
		log.Printf("Notice: Master user check error: %v", err)
	}

	teamService := service.NewTeamService(teamRepository, userRepository, 5)
	cashfreeService := service.NewCashfreeService(applicationConfig, teamRepository, userRepository)

	// 5. Initialize Handlers
	userHandler := handlers.NewUserHandler(userService, applicationConfig.JWTSecret)
	teamHandler := handlers.NewTeamHandler(teamService, cashfreeService)

	// 6. Initialize Fiber web application
	fiberApplication := fiber.New(fiber.Config{
		AppName:      "TransfiNITTe 2025 Backend API",
		ServerHeader: "Fiber",
	})

	// 7. Middlewares
	fiberApplication.Use(recover.New())
	fiberApplication.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	fiberApplication.Use(cors.New(cors.Config{
		AllowOrigins:     applicationConfig.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-User-ID, X-User-Email, Cookie",
		ExposeHeaders:    "Set-Cookie",
		AllowCredentials: true,
	}))

	// 8. Register all application routes
	routes.SetupRoutes(fiberApplication, userHandler, teamHandler, applicationConfig.JWTSecret)

	// 9. Start listening on configured port
	listenAddress := fmt.Sprintf(":%s", applicationConfig.Port)
	log.Printf("Server starting on port %s...", applicationConfig.Port)

	serverError := fiberApplication.Listen(listenAddress)
	if serverError != nil {
		log.Fatalf("Fatal: Server failed to start: %v", serverError)
	}
}
