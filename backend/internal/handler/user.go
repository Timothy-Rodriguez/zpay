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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	ctx, span := u.App.Tracer.Start(c.Request.Context(), "user.create")
	defer span.End()

	var createUserRequest model.CreateUserRequest

	if err := c.ShouldBindJSON(&createUserRequest); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		u.App.Logger.Warn("create_user_invalid_body",
			"http_path", c.FullPath(),
			"error", err.Error(),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	createUserRequest.Email = strings.TrimSpace(createUserRequest.Email)
	span.SetAttributes(
		attribute.String("user.email", createUserRequest.Email),
	)
	if createUserRequest.Email == "" {
		span.SetStatus(codes.Error, "missing email")
		u.App.Logger.Warn("create_user_missing_email",
			"http_path", c.FullPath(),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "email is required",
		})
		return
	}

	hashCtx, hashSpan := u.App.Tracer.Start(ctx, "user.hash_password")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(createUserRequest.Password), bcrypt.DefaultCost)
	if err != nil {
		hashSpan.RecordError(err)
		hashSpan.SetStatus(codes.Error, "password hash failed")
		hashSpan.End()

		span.RecordError(err)
		span.SetStatus(codes.Error, "password hash failed")

		u.App.Logger.Error("create_user_hash_failed",
			"email", createUserRequest.Email,
			"error", err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to hash password",
		})
		return
	}
	hashSpan.End()

	dbCtx, dbSpan := u.App.Tracer.Start(hashCtx, "db.create_user")
	if err := u.App.DB.CreateUser(dbCtx, createUserRequest.Email, string(hashedPassword)); err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "db create user failed")
		dbSpan.End()

		span.RecordError(err)
		span.SetStatus(codes.Error, "db create user failed")

		u.App.Logger.Error("create_user_db_failed",
			"email", createUserRequest.Email,
			"error", err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}
	dbSpan.End()

	// Set default balance
	_, balanceSpan := u.App.Tracer.Start(dbCtx, "db.init_user_balance")
	defaultBalance, _ := decimal.NewFromString("1000")
	if err := u.App.DB.UpdateBalace(dbCtx, createUserRequest.Email, defaultBalance); err != nil {
		balanceSpan.RecordError(err)
		balanceSpan.SetStatus(codes.Error, "init balance failed")
		balanceSpan.End()

		span.RecordError(err)
		span.SetStatus(codes.Error, "init balance failed")

		u.App.Logger.Error("create_user_balance_init_failed",
			"email", createUserRequest.Email,
			"error", err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "user created but failed to initialize account balance",
		})
		return
	}
	balanceSpan.End()

	span.SetStatus(codes.Ok, "user created")
	u.App.Logger.Info("create_user_success",
		"email", createUserRequest.Email,
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "user created successfully",
	})
}

func (u *UserHandler) LoginUser(c *gin.Context) {
	ctx, span := u.App.Tracer.Start(c.Request.Context(), "user.login")
	defer span.End()

	var loginRequest model.LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		u.App.Logger.Warn("login_user_invalid_body",
			"http_path", c.FullPath(),
			"error", err.Error(),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	span.SetAttributes(
		attribute.String("user.email", loginRequest.Email),
	)

	userClaims := make(map[string]interface{})
	userClaims[constants.ClaimsEmail] = loginRequest.Email

	accessToken, err := u.App.JWT.GenerateToken(userClaims, time.Second*5)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		u.App.Logger.Warn("error_creating_token",
			"http_path", c.FullPath(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}

	refreshClaims := make(map[string]interface{})
	refreshClaims[constants.ClaimsEmail] = loginRequest.Email
	refreshToken, err := u.App.JWT.GenerateToken(refreshClaims, time.Minute*30) // 30 minutes for testing
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		u.App.Logger.Warn("error_creating_token",
			"http_path", c.FullPath(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}

	// Check login and store refresh token
	_, dbSpan := u.App.Tracer.Start(ctx, "db.login_user")
	var loggedIn bool
	if loggedIn, err = u.App.DB.CheckLoginAndStoreRefreshToken(context.Background(), loginRequest.Email, loginRequest.Password, refreshToken); err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "db error while login")
		dbSpan.End()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	if !loggedIn {
		dbSpan.End()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "incorrect id/password",
		})
		return
	}
	dbSpan.End()

	u.setCookie(c, constants.JwtToken, accessToken, time.Second*5)
	u.setCookie(c, constants.RefrestToken, refreshToken, time.Minute*30) // 30 minutes for testing

	span.SetStatus(codes.Ok, "user logged in")
	u.App.Logger.Info("user_logged_in",
		"email", loginRequest.Email,
	)
	c.JSON(http.StatusOK, gin.H{
		"status":        "logged in",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
	})
}

// Logout endpoint - clears cookies and invalidates refresh token
func (u *UserHandler) LogoutUser(c *gin.Context) {
	// Get email from JWT claims (set by auth middleware)
	email, exists := c.Get(constants.ClaimsEmail)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	emailStr := email.(string)

	// Clear refresh token in database (invalidate it)
	if err := u.App.DB.ClearRefreshToken(context.Background(), emailStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to logout",
		})
		return
	}

	// Clear cookies
	u.clearCookie(c, constants.JwtToken)
	u.clearCookie(c, constants.RefrestToken)

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}

// Refresh endpoint - validates refresh token and returns new access token
func (u *UserHandler) RefreshToken(c *gin.Context) {

	refreshToken, err := c.Cookie(constants.RefrestToken)
	if err != nil {
		u.App.Logger.Error("refresh_no_cookie",
			"error", err.Error(),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "refresh token not provided",
		})
		return
	}
	u.App.Logger.Info("refresh_token_received",
		"token_length", len(refreshToken),
	)

	// Validate refresh token
	claims, err := u.App.JWT.ValidateToken(refreshToken)
	if err != nil {
		u.App.Logger.Error("refresh_validation_failed",
			"error", err.Error(),
		)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or expired refresh token",
		})
		return
	}

	// Extract email from claims
	email, ok := claims[constants.ClaimsEmail].(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token claims",
		})
		return
	}

	// Verify refresh token exists in database (not invalidated)
	storedToken, err := u.App.DB.GetRefreshToken(context.Background(), email)
	if err != nil || storedToken != refreshToken {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "refresh token has been invalidated",
		})
		return
	}

	// Generate new access token
	accessTokenClaims := make(map[string]interface{})
	accessTokenClaims[constants.ClaimsEmail] = email

	newAccessToken, err := u.App.JWT.GenerateToken(accessTokenClaims, time.Second*5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate access token",
		})
		return
	}

	// Set cookie if client is browser-based
	u.setCookie(c, constants.JwtToken, newAccessToken, time.Second*5)

	c.JSON(http.StatusOK, gin.H{
		"access_token": newAccessToken,
		"token_type":   "Bearer",
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

// CheckUserExists handler check if user exists
func (u *UserHandler) CheckUserExists(c *gin.Context) {
	ctx, span := u.App.Tracer.Start(c.Request.Context(), "user.check_exists")
	defer span.End()

	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		span.SetStatus(codes.Error, "missing email query param")
		u.App.Logger.Warn("check_user_exists_missing_email",
			"http_path", c.FullPath(),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "email query parameter is required"})
		return
	}

	span.SetAttributes(attribute.String("user.email", email))

	exists, err := u.App.DB.UserExists(ctx, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "db user_exists failed")
		u.App.Logger.Error("check_user_exists_db_error",
			"email", email,
			"error", err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check user"})
		return
	}

	span.SetAttributes(attribute.Bool("user.exists", exists))
	span.SetStatus(codes.Ok, "user_exists check success")

	u.App.Logger.Info("check_user_exists",
		"email", email,
		"exists", exists,
	)

	c.JSON(http.StatusOK, gin.H{"exists": exists})
}

// GET /get-accounts — returns up to 5 accounts with their balances (public, for testing)
func (u *UserHandler) GetAccounts(c *gin.Context) {
	ctx, span := u.App.Tracer.Start(c.Request.Context(), "user.get_accounts")
	defer span.End()

	accounts, err := u.App.DB.GetAccounts(ctx, 5)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "db get_accounts failed")
		u.App.Logger.Error("get_accounts_db_error", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch accounts"})
		return
	}

	span.SetStatus(codes.Ok, "get_accounts success")
	u.App.Logger.Info("get_accounts", "count", len(accounts))

	c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}
