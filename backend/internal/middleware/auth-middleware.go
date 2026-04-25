package middleware

import (
	"net/http"
	"strings"
	"zpay/internal/constants"
	"zpay/internal/model"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	App *model.App
}

func NewAuthHndler(app *model.App) *AuthHandler {
	return &AuthHandler{
		App: app,
	}
}

func (a *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		var accessToken string

		authHeader := strings.TrimSpace(c.GetHeader(constants.AuthorizationHeader))
		if token, found := strings.CutPrefix(authHeader, "Bearer "); found {
			accessToken = strings.TrimSpace(token)
		}

		claimsInterface, err := a.App.JWT.ValidateToken(accessToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "error parsing token",
			})
			c.Abort()
			return
		}

		c.Set(constants.Claims, claimsInterface)
		c.Set(constants.ClaimsEmail, claimsInterface[constants.ClaimsEmail].(string))

		c.Next()
	}
}
