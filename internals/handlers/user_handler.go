package handlers

import (
	"strings"
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

	// Generate a cryptographically signed JWT token and set HTTP-only cookie
	tokenString := userHandler.generateToken(syncedUserRecord)
	if tokenString != "" {
		userHandler.setAuthCookie(requestContext, tokenString)
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "User synchronized successfully",
		"data":    syncedUserRecord,
		"token":   tokenString,
	})
}

func (userHandler *UserHandler) HandleEmailLogin(requestContext *fiber.Ctx) error {
	var loginReq models.EmailLoginRequest
	if err := requestContext.BodyParser(&loginReq); err != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request payload: " + err.Error(),
		})
	}

	userRecord, err := userHandler.userService.LoginWithEmail(requestContext.Context(), loginReq)
	if err != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	// Generate a cryptographically signed JWT token and set HTTP-only cookie
	tokenString := userHandler.generateToken(userRecord)
	if tokenString != "" {
		userHandler.setAuthCookie(requestContext, tokenString)
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Login successful",
		"data":    userRecord,
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

	// Generate a cryptographically signed JWT token and set HTTP-only cookie
	tokenString := userHandler.generateToken(syncedUserRecord)
	if tokenString != "" {
		userHandler.setAuthCookie(requestContext, tokenString)
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "DAuth login successful",
		"data":    syncedUserRecord,
		"token":   tokenString,
	})
}

func (userHandler *UserHandler) generateToken(user *models.User) string {
	if user == nil {
		return ""
	}
	secret := userHandler.jwtSecret
	if secret == "" {
		secret = "tf-register-secret-key-2025"
	}
	claims := jwt.MapClaims{
		"sub":   user.UserID.String(),
		"email": user.Email,
		"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return ""
	}
	return signed
}

func (userHandler *UserHandler) setAuthCookie(requestContext *fiber.Ctx, tokenString string) {
	if tokenString == "" {
		return
	}

	isHTTPS := requestContext.Protocol() == "https" ||
		requestContext.Get("X-Forwarded-Proto") == "https" ||
		strings.HasPrefix(requestContext.Get("Origin"), "https://") ||
		strings.HasPrefix(requestContext.Get("Referer"), "https://")

	sameSite := "Lax"
	secure := false
	if isHTTPS {
		sameSite = "None"
		secure = true
	}

	requestContext.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
	})

	requestContext.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    tokenString,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
	})
}

func (userHandler *UserHandler) HandleLogout(requestContext *fiber.Ctx) error {
	isHTTPS := requestContext.Protocol() == "https" ||
		requestContext.Get("X-Forwarded-Proto") == "https" ||
		strings.HasPrefix(requestContext.Get("Origin"), "https://") ||
		strings.HasPrefix(requestContext.Get("Referer"), "https://")

	sameSite := "Lax"
	secure := false
	if isHTTPS {
		sameSite = "None"
		secure = true
	}

	requestContext.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
	})
	requestContext.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
	})

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Logged out successfully",
	})
}


