package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const UserIDKey = "userID"

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		tokenString := ""
		if strings.HasPrefix(header, "Bearer ") {
			tokenString = strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		} else if c.Request.URL.Path == "/ws/chat" {
			tokenString = strings.TrimSpace(c.Query("token"))
		}
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		claims := jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !token.Valid || claims.Subject == "" || claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(UserIDKey, claims.Subject)
		c.Next()
	}
}
