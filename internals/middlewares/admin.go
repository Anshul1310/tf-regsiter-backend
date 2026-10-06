package middlewares

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func extractAdminToken(c *fiber.Ctx) string {
	authHeader := c.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	adminCookie := c.Cookies("admin_token")
	if adminCookie != "" {
		return adminCookie
	}

	tokenCookie := c.Cookies("token")
	if tokenCookie != "" {
		return tokenCookie
	}

	// Fallback to raw Cookie header parsing
	cookieHeader := c.Get("Cookie")
	if cookieHeader != "" {
		for _, part := range strings.Split(cookieHeader, ";") {
			trimmed := strings.TrimSpace(part)
			if strings.HasPrefix(trimmed, "admin_token=") {
				return strings.TrimPrefix(trimmed, "admin_token=")
			}
			if strings.HasPrefix(trimmed, "token=") {
				return strings.TrimPrefix(trimmed, "token=")
			}
		}
	}

	return ""
}

func AuthenticateAdmin(jwtSecretKey string) fiber.Handler {
	if jwtSecretKey == "" {
		jwtSecretKey = "tf-register-secret-key-2025"
	}

	return func(c *fiber.Ctx) error {
		tokenString := extractAdminToken(c)

		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized: admin authentication required",
			})
		}

		claims := jwt.MapClaims{}
		parsedToken, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecretKey), nil
		})

		if err != nil || !parsedToken.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Invalid or expired admin session token",
			})
		}

		subClaim, hasSub := claims["sub"].(string)
		if !hasSub || subClaim == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Invalid token claims",
			})
		}

		adminUUID, err := uuid.Parse(subClaim)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Invalid admin identifier in token",
			})
		}

		email, _ := claims["email"].(string)
		name, _ := claims["name"].(string)
		role, _ := claims["role"].(string)
		if role == "" {
			role = "admin"
		}

		canUsers, _ := claims["can_manage_users"].(bool)
		canTeams, _ := claims["can_manage_teams"].(bool)
		canPayments, _ := claims["can_manage_payments"].(bool)
		canAdmins, _ := claims["can_manage_admins"].(bool)

		// Superadmin inherits all permissions
		if role == "superadmin" {
			canUsers = true
			canTeams = true
			canPayments = true
			canAdmins = true
		}

		c.Locals("adminID", adminUUID)
		c.Locals("adminEmail", email)
		c.Locals("adminName", name)
		c.Locals("adminRole", role)
		c.Locals("canManageUsers", canUsers)
		c.Locals("canManageTeams", canTeams)
		c.Locals("canManagePayments", canPayments)
		c.Locals("canManageAdmins", canAdmins)

		return c.Next()
	}
}

func RequireAdminPermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("adminRole").(string)
		if role == "superadmin" {
			return c.Next()
		}

		switch permission {
		case "users":
			if allowed, ok := c.Locals("canManageUsers").(bool); ok && allowed {
				return c.Next()
			}
		case "teams":
			if allowed, ok := c.Locals("canManageTeams").(bool); ok && allowed {
				return c.Next()
			}
		case "payments":
			if allowed, ok := c.Locals("canManagePayments").(bool); ok && allowed {
				return c.Next()
			}
		case "admins":
			if allowed, ok := c.Locals("canManageAdmins").(bool); ok && allowed {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Permission denied: '%s' management rights required", permission),
		})
	}
}

func RequireSuperAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("adminRole").(string)
		canAdmins, _ := c.Locals("canManageAdmins").(bool)

		if role == "superadmin" || canAdmins {
			return c.Next()
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Access restricted to Super Administrators only",
		})
	}
}
