// Package middleware provides HTTP middleware for the ZenReply API.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kietle/zenreply/pkg/response"
)

const (
	// ContextKeyUserID is the context key for the authenticated user ID.
	ContextKeyUserID = "user_id"
	// ContextKeySlackUserID is the context key for the Slack user ID.
	ContextKeySlackUserID = "slack_user_id"
)

// Claims represents the JWT token claims.
type Claims struct {
	UserID      string `json:"user_id"`
	SlackUserID string `json:"slack_user_id"`
	jwt.RegisteredClaims
}

// Auth returns a Gin middleware that validates JWT Bearer tokens.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			response.Unauthorized(c, "invalid authorization header format, expected: Bearer <token>")
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeySlackUserID, claims.SlackUserID)
		c.Next()
	}
}
