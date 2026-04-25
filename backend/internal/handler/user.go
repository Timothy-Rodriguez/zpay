package handler

import (
	"context"
	"net/http"
	"strings"
	"time"
	"zpay/internal/constants"
	"zpay/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	App *model.App
}

func NewUserHandler(app *model.App) *UserHandler {
	return &UserHandler{
		App: app,
	}
}

func (u *UserHandler) CreateUser(c *gin.Context) {
	var createUserRequest model.CreateUserRequest

	if err := c.ShouldBindJSON(&createUserRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	createUserRequest.Email = strings.TrimSpace(createUserRequest.Email)
	if createUserRequest.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "email is required",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(createUserRequest.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to hash password",
		})
		return
	}

	if err := u.App.DB.CreateUser(createUserRequest.Email, string(hashedPassword)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	// Set default balance
	defaultBalance, _ := decimal.NewFromString("1000")
	if err := u.App.DB.UpdateBalace(createUserRequest.Email, defaultBalance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "user created but failed to initialize account balance",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user created successfully",
	})
}

func (u *UserHandler) LoginUser(c *gin.Context) {
	var loginRequest model.LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// passwordByte := []byte(loginRequest.Password)
	// hashedPassword, err := bcrypt.GenerateFromPassword(passwordByte, bcrypt.DefaultCost)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{})
	// 	return
	// }

	// dbPassword, err := u.App.DB.GetUserPassword(loginRequest.Email)
	// if dbPassword != string(hashedPassword) {
	// 	c.JSON(http.StatusOK, gin.H{
	// 		"error": "incorrect email or password",
	// 	})
	// 	return
	// }

	userClaims := make(map[string]interface{})
	userClaims[constants.ClaimsEmail] = loginRequest.Email

	accessToken, err := u.App.JWT.GenerateToken(userClaims, time.Minute*5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}

	refreshToken, err := u.App.JWT.GenerateToken(userClaims, time.Minute*30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}

	// Check login and store refresh token
	var loggedIn bool
	if loggedIn, err = u.App.DB.CheckLoginAndStoreRefreshToken(context.Background(), loginRequest.Email, loginRequest.Password, refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	if !loggedIn {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "incorrect id/password",
		})
		return
	}

	u.setCookie(c, constants.JwtToken, accessToken, time.Minute*5)
	u.setCookie(c, constants.RefrestToken, refreshToken, time.Minute*30)

	c.JSON(http.StatusOK, gin.H{
		"status": "logged in",
	})
}

func (u *UserHandler) setCookie(c *gin.Context, name, value string, ttl time.Duration) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, int(ttl.Seconds()), "/", "", false, true)
}

func (u *UserHandler) clearCookie(c *gin.Context, name string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, "", -1, "/", "", false, true)
}
