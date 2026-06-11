package httpapi

import (
	"net/http"
	"strings"

	"businessapp/backend/internal/auth"
	"businessapp/backend/internal/model"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const userIDKey = "userID"
const accountIDKey = "accountID"
const currentUserKey = "currentUser"

func authMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
			return
		}
		rawID, err := auth.ParseToken(secret, strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		id, err := primitive.ObjectIDFromHex(rawID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			return
		}
		c.Set(userIDKey, id)
		c.Next()
	}
}

func cors(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func currentUserID(c *gin.Context) primitive.ObjectID {
	return c.MustGet(userIDKey).(primitive.ObjectID)
}

func currentAccountID(c *gin.Context) primitive.ObjectID {
	return c.MustGet(accountIDKey).(primitive.ObjectID)
}

func currentUser(c *gin.Context) model.User {
	return c.MustGet(currentUserKey).(model.User)
}
