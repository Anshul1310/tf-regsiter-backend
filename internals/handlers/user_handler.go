package handlers

import (
	"time"

	"github.com/Anshul1310/tf-register/internals/models"
	"github.com/Anshul1310/tf-register/internals/service"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserHandler struct {
	userService *service.UserService
	jwtSecret   string
}

func NewUserHandler(userService *service.UserService, jwtSecret string) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

func (userHandler *UserHandler) GetCurrentUser(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}

	authenticatedUserID := authenticatedUserValue.(uuid.UUID)
	userProfile, retrieveError := userHandler.userService.GetUserProfile(requestContext.Context(), authenticatedUserID)
	if retrieveError != nil {
		return requestContext.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": retrieveError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    userProfile,
	})
}

func (userHandler *UserHandler) GetUserByID(requestContext *fiber.Ctx) error {
	userIdentifierParameter := requestContext.Params("id")
	parsedUserUUID, parseError := uuid.Parse(userIdentifierParameter)
	if parseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID format: must be a valid UUID",
		})
	}

	userProfile, retrieveError := userHandler.userService.GetUserProfile(requestContext.Context(), parsedUserUUID)
	if retrieveError != nil {
		return requestContext.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": retrieveError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    userProfile,
	})
}

func (userHandler *UserHandler) SyncUser(requestContext *fiber.Ctx) error {
	var syncUserRequest models.SyncUserRequest
	bodyParseError := requestContext.BodyParser(&syncUserRequest)
	if bodyParseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse request payload: " + bodyParseError.Error(),
		})
	}

	// If user_id is not in payload, check authenticated context
	if syncUserRequest.UserID == uuid.Nil {
		authenticatedUserValue := requestContext.Locals("userID")
		if authenticatedUserValue != nil {
			syncUserRequest.UserID = authenticatedUserValue.(uuid.UUID)
		}
	}

	if syncUserRequest.UserID == uuid.Nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User ID is required",
		})
	}

	syncedUserRecord, syncError := userHandler.userService.SyncUser(requestContext.Context(), syncUserRequest)
	if syncError != nil {
		return requestContext.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": syncError.Error(),
		})
	}

	// Generate a cryptographically signed JWT token
	var tokenString string
	if userHandler.jwtSecret != "" {
		claims := jwt.MapClaims{
			"sub":   syncedUserRecord.UserID.String(),
			"email": syncedUserRecord.Email,
			"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(),
			"iat":   time.Now().Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(userHandler.jwtSecret))
		if err == nil {
			tokenString = signed
		}
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "User synchronized successfully",
		"data":    syncedUserRecord,
		"token":   tokenString,
	})
}

func (userHandler *UserHandler) UpdateProfile(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}

	authenticatedUserID := authenticatedUserValue.(uuid.UUID)

	var profileUpdateRequest models.UpdateUserProfileRequest
	bodyParseError := requestContext.BodyParser(&profileUpdateRequest)
	if bodyParseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse profile update request: " + bodyParseError.Error(),
		})
	}

	updatedUserProfile, updateError := userHandler.userService.UpdateProfile(
		requestContext.Context(),
		authenticatedUserID,
		profileUpdateRequest,
	)
	if updateError != nil {
		return requestContext.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": updateError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Profile updated successfully",
		"data":    updatedUserProfile,
	})
}

func (userHandler *UserHandler) LeaveTeam(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}

	authenticatedUserID := authenticatedUserValue.(uuid.UUID)
	leaveError := userHandler.userService.LeaveTeam(requestContext.Context(), authenticatedUserID)
	if leaveError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": leaveError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Successfully left the team",
	})
}

func (userHandler *UserHandler) HandleDAuthLogin(requestContext *fiber.Ctx) error {
	var dauthReq models.DAuthLoginRequest
	bodyParseError := requestContext.BodyParser(&dauthReq)
	if bodyParseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse DAuth request payload: " + bodyParseError.Error(),
		})
	}

	if dauthReq.Code == "" {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Authorization code is required",
		})
	}

	syncedUserRecord, dauthError := userHandler.userService.LoginWithDAuth(requestContext.Context(), dauthReq)
	if dauthError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": dauthError.Error(),
		})
	}

	// Generate a cryptographically signed JWT token
	var tokenString string
	if userHandler.jwtSecret != "" {
		claims := jwt.MapClaims{
			"sub":   syncedUserRecord.UserID.String(),
			"email": syncedUserRecord.Email,
			"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(),
			"iat":   time.Now().Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(userHandler.jwtSecret))
		if err == nil {
			tokenString = signed
		}
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "DAuth login successful",
		"data":    syncedUserRecord,
		"token":   tokenString,
	})
}

