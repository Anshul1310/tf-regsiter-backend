package routes

import (
	"github.com/Anshul1310/tf-register/internals/handlers"
	"github.com/Anshul1310/tf-register/internals/middlewares"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(
	fiberApplication *fiber.App,
	userHandler *handlers.UserHandler,
	teamHandler *handlers.TeamHandler,
	jwtSecretKey string,
) {
	// Root health check endpoint
	fiberApplication.Get("/", func(requestContext *fiber.Ctx) error {
		return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"service": "TransfiNITTe 2025 Backend API",
			"status":  "healthy",
		})
	})

	fiberApplication.Get("/health", func(requestContext *fiber.Ctx) error {
		return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "UP",
		})
	})

	// Setup route groups on both "/" and "/api" for frontend compatibility
	routePrefixes := []string{"", "/api"}

	for _, prefix := range routePrefixes {
		rootGroup := fiberApplication.Group(prefix)

		// -------------------------------------------------------------
		// User Routes
		// -------------------------------------------------------------
		userGroup := rootGroup.Group("/user")

		// Public/Semi-public user endpoints
		userGroup.Post("/dauth", userHandler.HandleDAuthLogin)
		userGroup.Post("/login", userHandler.HandleEmailLogin)
		userGroup.Post("/sync", middlewares.OptionalAuthenticateUser(jwtSecretKey), userHandler.SyncUser)
		userGroup.Get("/:id", userHandler.GetUserByID)
		userGroup.Post("/logout", userHandler.HandleLogout)

		// Authenticated user endpoints
		userGroup.Get("/me", middlewares.AuthenticateUser(jwtSecretKey), userHandler.GetCurrentUser)
		userGroup.Post("/profile", middlewares.AuthenticateUser(jwtSecretKey), userHandler.UpdateProfile)
		userGroup.Put("/profile", middlewares.AuthenticateUser(jwtSecretKey), userHandler.UpdateProfile)
		userGroup.Post("/leave-team", middlewares.AuthenticateUser(jwtSecretKey), userHandler.LeaveTeam)

		// -------------------------------------------------------------
		// Team Routes
		// -------------------------------------------------------------
		teamGroup := rootGroup.Group("/team")

		// Public team endpoints
		teamGroup.Get("/public", teamHandler.GetAllPublicTeams)
		teamGroup.Get("/stats/payment", teamHandler.GetPaymentStats)
		teamGroup.Get("/:id", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.GetTeamByID)
		teamGroup.Get("/:id/members/count", teamHandler.GetMemberCount)

		// Team joining and creation
		teamGroup.Post("/create", middlewares.OptionalAuthenticateUser(jwtSecretKey), teamHandler.CreateTeam)
		teamGroup.Put("/create", middlewares.OptionalAuthenticateUser(jwtSecretKey), teamHandler.CreateTeam)
		teamGroup.Post("/join", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.JoinTeam)
		teamGroup.Post("/apply", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.ApplyToJoinTeam)

		// Team join requests (leader actions)
		teamGroup.Get("/:teamId/requests", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.GetTeamJoinRequests)
		teamGroup.Post("/:teamId/requests/:requestId/accept", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.AcceptJoinRequest)
		teamGroup.Post("/:teamId/requests/:requestId/reject", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.RejectJoinRequest)

		// Team modification (leader actions)
		teamGroup.Patch("/:teamId/visibility", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.UpdateVisibility)
		teamGroup.Patch("/:teamId/visiblity", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.UpdateVisibility)
		teamGroup.Patch("/:teamId/name", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.UpdateTeamName)
		teamGroup.Patch("/:teamName/name", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.UpdateTeamName)
		teamGroup.Patch("/:teamId/details", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.UpdateTeamDetails)
		teamGroup.Post("/:teamId/regenerate-id", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.RegenerateTeamID)
		teamGroup.Delete("/:teamId", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.DeleteTeam)
		teamGroup.Delete("/delete", middlewares.AuthenticateUser(jwtSecretKey), teamHandler.DeleteTeam)

		// Manual payment fallback endpoint
		teamGroup.Post("/:teamId/pay", teamHandler.SubmitManualPayment)

		// Cashfree checkout & payment endpoints
		rootGroup.Post("/checkout", teamHandler.CreateCheckoutOrder)
		rootGroup.Post("/payment/checkout", teamHandler.CreateCheckoutOrder)
		rootGroup.Post("/payment/verify", teamHandler.VerifyPayment)
		rootGroup.Post("/payment/webhook", teamHandler.HandleCashfreeWebhook)
	}
}
