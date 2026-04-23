package handler

import (
	"net/http"
	"strings"
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
	var req model.CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "email is required",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to hash password",
		})
		return
	}

	if err := u.App.DB.CreateUser(req.Email, string(hashedPassword)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	// Set default balance
	defaultBalance, _ := decimal.NewFromString("1000")
	if err := u.App.DB.UpdateBalace(req.Email, defaultBalance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "user created but failed to initialize account balance",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user created successfully",
	})
}
