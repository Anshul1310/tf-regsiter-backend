package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/Anshul1310/tf-register/internals/models"
	"github.com/Anshul1310/tf-register/internals/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AdminHandler struct {
	adminService *service.AdminService
	jwtSecret    string
}

func NewAdminHandler(adminService *service.AdminService, jwtSecret string) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
		jwtSecret:    jwtSecret,
	}
}

// -------------------------------------------------------------
// Auth Endpoints
// -------------------------------------------------------------

func (h *AdminHandler) HandleGoogleLogin(c *fiber.Ctx) error {
	var req models.GoogleAuthRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   err.Error(),
		})
	}

	token := req.Credential
	if token == "" {
		token = req.AccessToken
	}
	if token == "" {
		token = req.Code
	}

	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Google token or credential is required",
		})
	}

	jwtToken, adminUser, err := h.adminService.HandleGoogleLogin(c.Context(), token)
	if err != nil {
		status := fiber.StatusUnauthorized
		if err == service.ErrAdminUnauthorized {
			status = fiber.StatusForbidden
			return c.Status(status).JSON(models.APIResponse{
				Success: false,
				Message: "Access Denied: Your email is not registered as an authorized administrator. Please ask a Super Admin to add your email to the admin panel.",
				Error:   err.Error(),
			})
		}
		return c.Status(status).JSON(models.APIResponse{
			Success: false,
			Message: err.Error(),
			Error:   err.Error(),
		})
	}

	// Set cookie
	c.Cookie(&fiber.Cookie{
		Name:     "admin_token",
		Value:    jwtToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   false, // set true in https prod
	})

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Admin authenticated successfully",
		Data: fiber.Map{
			"token": jwtToken,
			"admin": adminUser,
		},
	})
}

func (h *AdminHandler) HandleDevLogin(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil || req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Email is required",
		})
	}

	jwtToken, adminUser, err := h.adminService.HandleDevLogin(c.Context(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "admin_token",
		Value:    jwtToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Dev admin session created",
		Data: fiber.Map{
			"token": jwtToken,
			"admin": adminUser,
		},
	})
}

func (h *AdminHandler) GetAdminMe(c *fiber.Ctx) error {
	adminID, ok := c.Locals("adminID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(models.APIResponse{
			Success: false,
			Message: "Unauthorized",
		})
	}

	adminUser, err := h.adminService.GetAdminByID(c.Context(), adminID)
	if err != nil || adminUser == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Admin profile not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Admin profile retrieved",
		Data:    adminUser,
	})
}

func (h *AdminHandler) HandleLogout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "admin_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

// -------------------------------------------------------------
// Dashboard Analytics
// -------------------------------------------------------------

func (h *AdminHandler) GetDashboardStats(c *fiber.Ctx) error {
	stats, err := h.adminService.GetDashboardStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to fetch dashboard statistics",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Dashboard statistics retrieved",
		Data:    stats,
	})
}

// -------------------------------------------------------------
// User Management Handlers
// -------------------------------------------------------------

func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	search := c.Query("search", "")
	hostel := c.Query("hostel", "")
	mess := c.Query("mess", "")
	gender := c.Query("gender", "")
	hasTeam := c.Query("has_team", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	users, totalCount, err := h.adminService.ListUsers(c.Context(), search, hostel, mess, gender, hasTeam, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to list users",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Users retrieved successfully",
		Data: fiber.Map{
			"users": users,
			"pagination": fiber.Map{
				"total": totalCount,
				"page":  page,
				"limit": limit,
				"pages": (totalCount + limit - 1) / limit,
			},
		},
	})
}

func (h *AdminHandler) GetUserByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	userUUID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid user UUID",
		})
	}

	user, team, err := h.adminService.GetUserByID(c.Context(), userUUID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "User not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "User details retrieved",
		Data: fiber.Map{
			"user": user,
			"team": team,
		},
	})
}

func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	userUUID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid user UUID",
		})
	}

	var req models.UpdateUserAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid request payload",
			Error:   err.Error(),
		})
	}

	if err := h.adminService.UpdateUser(c.Context(), userUUID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to update user",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "User updated successfully",
	})
}

func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	userUUID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid user UUID",
		})
	}

	if err := h.adminService.DeleteUser(c.Context(), userUUID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to delete user",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "User deleted successfully",
	})
}

// -------------------------------------------------------------
// Team Management Handlers
// -------------------------------------------------------------

func (h *AdminHandler) ListTeams(c *fiber.Ctx) error {
	search := c.Query("search", "")
	domain := c.Query("domain", "")
	paymentStatus := c.Query("payment_status", "")
	isPublic := c.Query("is_public", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	teams, totalCount, err := h.adminService.ListTeams(c.Context(), search, domain, paymentStatus, isPublic, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to list teams",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Teams retrieved successfully",
		Data: fiber.Map{
			"teams": teams,
			"pagination": fiber.Map{
				"total": totalCount,
				"page":  page,
				"limit": limit,
				"pages": (totalCount + limit - 1) / limit,
			},
		},
	})
}

func (h *AdminHandler) GetTeamByID(c *fiber.Ctx) error {
	teamID := c.Params("id")
	team, err := h.adminService.GetTeamByID(c.Context(), teamID)
	if err != nil || team == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
			Success: false,
			Message: "Team not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Team retrieved successfully",
		Data:    team,
	})
}

func (h *AdminHandler) CreateTeam(c *fiber.Ctx) error {
	var req models.Team
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid team data",
			Error:   err.Error(),
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Team name is required",
		})
	}

	if err := h.adminService.CreateTeam(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to create team",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Team created successfully",
		Data:    req,
	})
}

func (h *AdminHandler) UpdateTeam(c *fiber.Ctx) error {
	teamID := c.Params("id")
	var req models.UpdateTeamAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid update request",
			Error:   err.Error(),
		})
	}

	if err := h.adminService.UpdateTeam(c.Context(), teamID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to update team",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Team updated successfully",
	})
}

func (h *AdminHandler) UpdateTeamPaymentStatus(c *fiber.Ctx) error {
	teamID := c.Params("id")
	var req struct {
		PaymentStatus string `json:"payment_status"`
	}
	if err := c.BodyParser(&req); err != nil || req.PaymentStatus == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Payment status is required",
		})
	}

	if err := h.adminService.UpdateTeamPaymentStatus(c.Context(), teamID, req.PaymentStatus); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to update payment status",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Team payment status updated successfully",
	})
}

func (h *AdminHandler) DeleteTeam(c *fiber.Ctx) error {
	teamID := c.Params("id")
	if err := h.adminService.DeleteTeam(c.Context(), teamID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to delete team",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Team deleted successfully",
	})
}

func (h *AdminHandler) AddTeamMember(c *fiber.Ctx) error {
	teamID := c.Params("id")
	var req models.AddTeamMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid member request",
			Error:   err.Error(),
		})
	}

	var targetUserID uuid.UUID
	if req.UserID != nil && *req.UserID != uuid.Nil {
		targetUserID = *req.UserID
	} else if req.Email != nil && *req.Email != "" {
		// Look up user by email
		users, _, err := h.adminService.ListUsers(c.Context(), *req.Email, "", "", "", "", 1, 1)
		if err != nil || len(users) == 0 {
			return c.Status(fiber.StatusNotFound).JSON(models.APIResponse{
				Success: false,
				Message: "No user found with the given email",
			})
		}
		targetUserID = users[0].UserID
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "User ID or Email is required",
		})
	}

	if err := h.adminService.AddTeamMember(c.Context(), teamID, targetUserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to add member to team",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Member added to team successfully",
	})
}

func (h *AdminHandler) RemoveTeamMember(c *fiber.Ctx) error {
	teamID := c.Params("id")
	userParam := c.Params("userId")
	userUUID, err := uuid.Parse(userParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid user UUID",
		})
	}

	if err := h.adminService.RemoveTeamMember(c.Context(), teamID, userUUID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to remove member from team",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Member removed from team successfully",
	})
}

func (h *AdminHandler) ChangeTeamLeader(c *fiber.Ctx) error {
	teamID := c.Params("id")
	var req models.ChangeTeamLeaderRequest
	if err := c.BodyParser(&req); err != nil || req.NewLeaderUserID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Valid new leader user_id is required",
		})
	}

	if err := h.adminService.ChangeTeamLeader(c.Context(), teamID, req.NewLeaderUserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to change team leader",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Team leader updated successfully",
	})
}

// -------------------------------------------------------------
// Payment Management Handlers
// -------------------------------------------------------------

func (h *AdminHandler) ListPayments(c *fiber.Ctx) error {
	search := c.Query("search", "")
	status := c.Query("status", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	payments, totalCount, err := h.adminService.ListPayments(c.Context(), search, status, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to list payments",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Payments retrieved successfully",
		Data: fiber.Map{
			"payments": payments,
			"pagination": fiber.Map{
				"total": totalCount,
				"page":  page,
				"limit": limit,
				"pages": (totalCount + limit - 1) / limit,
			},
		},
	})
}

func (h *AdminHandler) VerifyManualPayment(c *fiber.Ctx) error {
	var req models.ManualPaymentVerifyRequest
	if err := c.BodyParser(&req); err != nil || req.TeamID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Team ID and payment details are required",
		})
	}

	if req.PaymentStatus == "" {
		req.PaymentStatus = "SUCCESS"
	}

	if err := h.adminService.VerifyManualPayment(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to record manual payment verification",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Manual payment verified and team status updated",
	})
}

// -------------------------------------------------------------
// Admin & RBAC User Management Handlers
// -------------------------------------------------------------

func (h *AdminHandler) ListAdmins(c *fiber.Ctx) error {
	admins, err := h.adminService.ListAdmins(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to list administrators",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Admins retrieved successfully",
		Data:    admins,
	})
}

func (h *AdminHandler) CreateAdmin(c *fiber.Ctx) error {
	var req models.CreateAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid admin request",
			Error:   err.Error(),
		})
	}

	if strings.TrimSpace(req.Email) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Email is required to grant admin access",
		})
	}

	admin, err := h.adminService.CreateAdmin(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to add admin user",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(models.APIResponse{
		Success: true,
		Message: "Admin user added successfully",
		Data:    admin,
	})
}

func (h *AdminHandler) UpdateAdmin(c *fiber.Ctx) error {
	idParam := c.Params("id")
	adminUUID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid admin UUID",
		})
	}

	var req models.UpdateAdminRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid update request",
			Error:   err.Error(),
		})
	}

	if err := h.adminService.UpdateAdmin(c.Context(), adminUUID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to update admin",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Admin updated successfully",
	})
}

func (h *AdminHandler) DeleteAdmin(c *fiber.Ctx) error {
	idParam := c.Params("id")
	adminUUID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "Invalid admin UUID",
		})
	}

	// Prevent deleting yourself
	currentAdminID, _ := c.Locals("adminID").(uuid.UUID)
	if currentAdminID == adminUUID {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Message: "You cannot delete your own admin account",
		})
	}

	if err := h.adminService.DeleteAdmin(c.Context(), adminUUID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to delete admin",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.APIResponse{
		Success: true,
		Message: "Admin removed successfully",
	})
}

// -------------------------------------------------------------
// CSV Exports
// -------------------------------------------------------------

func (h *AdminHandler) ExportUsers(c *fiber.Ctx) error {
	csvBytes, err := h.adminService.ExportUsersCSV(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to export users",
			Error:   err.Error(),
		})
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", `attachment; filename="transfinitte_users.csv"`)
	return c.Send(csvBytes)
}

func (h *AdminHandler) ExportTeams(c *fiber.Ctx) error {
	csvBytes, err := h.adminService.ExportTeamsCSV(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to export teams",
			Error:   err.Error(),
		})
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", `attachment; filename="transfinitte_teams.csv"`)
	return c.Send(csvBytes)
}

func (h *AdminHandler) ExportPayments(c *fiber.Ctx) error {
	csvBytes, err := h.adminService.ExportPaymentsCSV(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Message: "Failed to export payments",
			Error:   err.Error(),
		})
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", `attachment; filename="transfinitte_payments.csv"`)
	return c.Send(csvBytes)
}
