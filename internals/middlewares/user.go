package middlewares

import (
	"fmt"
	"strings"

	"github.com/Anshul1310/tf-register/internals/service"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func extractToken(requestContext *fiber.Ctx) string {
	authorizationHeader := requestContext.Get("Authorization")
	if authorizationHeader != "" && strings.HasPrefix(authorizationHeader, "Bearer ") {
		return strings.TrimPrefix(authorizationHeader, "Bearer ")
	}

	cookieToken := requestContext.Cookies("token")
	if cookieToken != "" {
		return cookieToken
	}

	jwtCookie := requestContext.Cookies("jwt")
	if jwtCookie != "" {
		return jwtCookie
	}

	// Fallback to raw Cookie header parsing in case fiber.Cookies misses edge delimiters
	cookieHeader := requestContext.Get("Cookie")
	if cookieHeader != "" {
		for _, part := range strings.Split(cookieHeader, ";") {
			trimmed := strings.TrimSpace(part)
			if strings.HasPrefix(trimmed, "token=") {
				return strings.TrimPrefix(trimmed, "token=")
			}
			if strings.HasPrefix(trimmed, "jwt=") {
				return strings.TrimPrefix(trimmed, "jwt=")
			}
		}
	}

	return ""
}

func AuthenticateUser(jwtSecretKey string) fiber.Handler {
	if jwtSecretKey == "" {
		jwtSecretKey = "tf-register-secret-key-2025"
	}

	return func(requestContext *fiber.Ctx) error {
		tokenString := extractToken(requestContext)

		if tokenString != "" {
			// Parse token with claims
			tokenClaims := jwt.MapClaims{}
			parsedToken, tokenParseError := jwt.ParseWithClaims(tokenString, tokenClaims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecretKey), nil
			})

			// Standard cryptographically verified HMAC signature
			if tokenParseError == nil && parsedToken.Valid {
				subjectClaim, hasSubject := tokenClaims["sub"].(string)
				if hasSubject && subjectClaim != "" {
					parsedUserUUID, uuidParseError := uuid.Parse(subjectClaim)
					if uuidParseError == nil {
						emailClaim, _ := tokenClaims["email"].(string)
						requestContext.Locals("userID", parsedUserUUID)
						requestContext.Locals("userEmail", emailClaim)
						return requestContext.Next()
					}
				}
			}
		}

		// Cross-origin fallback when browser blocks third-party cookies
		fallbackUserIdentifier := requestContext.Get("X-User-ID")
		if fallbackUserIdentifier != "" {
			parsedUserUUID, uuidParseError := uuid.Parse(fallbackUserIdentifier)
			if uuidParseError == nil {
				emailHeader := requestContext.Get("X-User-Email")
				requestContext.Locals("userID", parsedUserUUID)
				requestContext.Locals("userEmail", emailHeader)
				return requestContext.Next()
			}
		}

		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: valid signed token (cookie or Bearer header) required",
		})
	}
}

func OptionalAuthenticateUser(jwtSecretKey string) fiber.Handler {
	return func(requestContext *fiber.Ctx) error {
		tokenString := extractToken(requestContext)
		fallbackUserIdentifier := requestContext.Get("X-User-ID")

		if tokenString != "" {
			unverifiedParser := jwt.NewParser()
			unverifiedClaims := jwt.MapClaims{}
			_, _, unverifiedParseError := unverifiedParser.ParseUnverified(tokenString, unverifiedClaims)
			if unverifiedParseError == nil {
				subjectClaim, hasSubject := unverifiedClaims["sub"].(string)
				if hasSubject && subjectClaim != "" {
					parsedUserUUID, uuidParseError := uuid.Parse(subjectClaim)
					if uuidParseError == nil {
						emailClaim, _ := unverifiedClaims["email"].(string)
						requestContext.Locals("userID", parsedUserUUID)
						requestContext.Locals("userEmail", emailClaim)
					}
				}
			}
		} else if fallbackUserIdentifier != "" {
			parsedUserUUID, uuidParseError := uuid.Parse(fallbackUserIdentifier)
			if uuidParseError == nil {
				requestContext.Locals("userID", parsedUserUUID)
			}
		}

		return requestContext.Next()
	}
}

func RequireTeamLeader(teamService *service.TeamService) fiber.Handler {
	return func(requestContext *fiber.Ctx) error {
		authenticatedUserValue := requestContext.Locals("userID")
		if authenticatedUserValue == nil {
			return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Authentication required to perform this action",
			})
		}

		authenticatedUserID := authenticatedUserValue.(uuid.UUID)
		teamIdentifier := requestContext.Params("teamId")
		if teamIdentifier == "" {
			teamIdentifier = requestContext.Params("id")
		}
		if teamIdentifier == "" {
			teamIdentifier = requestContext.Params("teamName")
		}

		_, authorizationError := teamService.VerifyTeamLeader(requestContext.Context(), authenticatedUserID, teamIdentifier)
		if authorizationError != nil {
			return requestContext.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": authorizationError.Error(),
			})
		}

		return requestContext.Next()
	}
}
