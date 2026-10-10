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
	adminHandler *handlers.AdminHandler,
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

		// Authenticated user endpoints
		userGroup.Get("/me", middlewares.AuthenticateUser(jwtSecretKey), userHandler.GetCurrentUser)
		userGroup.Post("/profile", middlewares.AuthenticateUser(jwtSecretKey), userHandler.UpdateProfile)
		userGroup.Put("/profile", middlewares.AuthenticateUser(jwtSecretKey), userHandler.UpdateProfile)
		userGroup.Post("/leave-team", middlewares.AuthenticateUser(jwtSecretKey), userHandler.LeaveTeam)

		// Public/Semi-public user endpoints
		userGroup.Post("/dauth", userHandler.HandleDAuthLogin)
		userGroup.Post("/login", userHandler.HandleEmailLogin)
		userGroup.Post("/sync", middlewares.OptionalAuthenticateUser(jwtSecretKey), userHandler.SyncUser)
		userGroup.Get("/:id", userHandler.GetUserByID)
		userGroup.Post("/logout", userHandler.HandleLogout)

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
		rootGroup.Get("/payment/config", teamHandler.GetPaymentConfig)
		rootGroup.Post("/checkout", teamHandler.CreateCheckoutOrder)
		rootGroup.Post("/payment/checkout", teamHandler.CreateCheckoutOrder)
		rootGroup.Post("/payment/verify", teamHandler.VerifyPayment)
		rootGroup.Post("/payment/webhook", teamHandler.HandleCashfreeWebhook)

		// -------------------------------------------------------------
		// Admin Portal Routes (RBAC Protected)
		// -------------------------------------------------------------
		adminGroup := rootGroup.Group("/admin")

		// Admin Auth Endpoints
		adminAuth := adminGroup.Group("/auth")
		adminAuth.Post("/google", adminHandler.HandleGoogleLogin)
		adminAuth.Post("/dev-login", adminHandler.HandleDevLogin)
		adminAuth.Post("/logout", adminHandler.HandleLogout)
		adminAuth.Get("/me", middlewares.AuthenticateAdmin(jwtSecretKey), adminHandler.GetAdminMe)

		// Admin Dashboard
		adminGroup.Get("/dashboard/stats", middlewares.AuthenticateAdmin(jwtSecretKey), adminHandler.GetDashboardStats)

		// Admin User Management
		adminUsers := adminGroup.Group("/users", middlewares.AuthenticateAdmin(jwtSecretKey), middlewares.RequireAdminPermission("users"))
		adminUsers.Get("/", adminHandler.ListUsers)
		adminUsers.Get("/export", adminHandler.ExportUsers)
		adminUsers.Get("/:id", adminHandler.GetUserByID)
		adminUsers.Put("/:id", adminHandler.UpdateUser)
		adminUsers.Delete("/:id", adminHandler.DeleteUser)

		// Admin Team Management
		adminTeams := adminGroup.Group("/teams", middlewares.AuthenticateAdmin(jwtSecretKey), middlewares.RequireAdminPermission("teams"))
		adminTeams.Get("/", adminHandler.ListTeams)
		adminTeams.Get("/export", adminHandler.ExportTeams)
		adminTeams.Get("/:id", adminHandler.GetTeamByID)
		adminTeams.Post("/", adminHandler.CreateTeam)
		adminTeams.Put("/:id", adminHandler.UpdateTeam)
		adminTeams.Patch("/:id/payment-status", adminHandler.UpdateTeamPaymentStatus)
		adminTeams.Post("/:id/members", adminHandler.AddTeamMember)
		adminTeams.Delete("/:id/members/:userId", adminHandler.RemoveTeamMember)
		adminTeams.Post("/:id/leader", adminHandler.ChangeTeamLeader)
		adminTeams.Delete("/:id", adminHandler.DeleteTeam)

		// Admin Payment Management
		adminPayments := adminGroup.Group("/payments", middlewares.AuthenticateAdmin(jwtSecretKey), middlewares.RequireAdminPermission("payments"))
		adminPayments.Get("/", adminHandler.ListPayments)
		adminPayments.Get("/export", adminHandler.ExportPayments)
		adminPayments.Post("/verify", adminHandler.VerifyManualPayment)

		// Admin & RBAC Roles Management
		adminAdmins := adminGroup.Group("/admins", middlewares.AuthenticateAdmin(jwtSecretKey), middlewares.RequireAdminPermission("admins"))
		adminAdmins.Get("/", adminHandler.ListAdmins)
		adminAdmins.Post("/", adminHandler.CreateAdmin)
		adminAdmins.Put("/:id", adminHandler.UpdateAdmin)
		adminAdmins.Delete("/:id", adminHandler.DeleteAdmin)
	}
}
