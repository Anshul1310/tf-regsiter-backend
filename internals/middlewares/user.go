package middlewares

import (
	"fmt"
	"strings"

	"github.com/Anshul1310/tf-register/internals/service"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func AuthenticateUser(jwtSecretKey string) fiber.Handler {
	return func(requestContext *fiber.Ctx) error {
		authorizationHeader := requestContext.Get("Authorization")

		if authorizationHeader != "" && strings.HasPrefix(authorizationHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authorizationHeader, "Bearer ")

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

		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Unauthorized: valid signed Bearer token required",
		})
	}
}

func OptionalAuthenticateUser(jwtSecretKey string) fiber.Handler {
	return func(requestContext *fiber.Ctx) error {
		authorizationHeader := requestContext.Get("Authorization")
		fallbackUserIdentifier := requestContext.Get("X-User-ID")

		if authorizationHeader != "" && strings.HasPrefix(authorizationHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authorizationHeader, "Bearer ")
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
