// Package identity is the HTTP adapter for auth, profile and preferences.
package identity

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	domain "github.com/mokchan/webnovel-backend/internal/domain/identity"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/middleware"
	identitysvc "github.com/mokchan/webnovel-backend/internal/service/identity"
)

// Handler exposes identity use cases over HTTP.
type Handler struct {
	Service *identitysvc.Service
}

// New wires a handler onto a service.
func New(svc *identitysvc.Service) *Handler { return &Handler{Service: svc} }

// Register mounts the auth and profile routes. authThrottle rate-limits the
// unauthenticated auth endpoints per IP (I-SEC-03).
func (h *Handler) Register(r gin.IRouter, requireAuth, authThrottle gin.HandlerFunc) {
	auth := r.Group("/auth", authThrottle)
	auth.POST("/register", h.register)
	auth.POST("/login", h.login)
	auth.POST("/refresh", h.refresh)
	auth.POST("/logout", h.logout)
	auth.GET("/me", requireAuth, h.me)

	users := r.Group("/users/me", requireAuth)
	users.GET("", h.me)
	users.PATCH("", h.updateMe)
	users.GET("/prefs", h.getPrefs)
	users.PUT("/prefs", h.putPrefs)
	users.GET("/genre-prefs", h.getGenrePrefs)
	users.PUT("/genre-prefs", h.putGenrePrefs)
}

func (h *Handler) register(c *gin.Context) {
	var body registerRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลสมัครสมาชิกไม่ถูกต้อง")
		return
	}

	pair, err := h.Service.Register(c.Request.Context(), domain.Registration{
		Username:    body.Username,
		Email:       body.Email,
		Password:    body.Password,
		DisplayName: body.DisplayName,
	}, c.Request.UserAgent())
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAuthResponse(*pair, true))
}

func (h *Handler) login(c *gin.Context) {
	var body loginRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลเข้าสู่ระบบไม่ถูกต้อง")
		return
	}

	pair, err := h.Service.Login(c.Request.Context(), body.Email, body.Password, c.Request.UserAgent())
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toAuthResponse(*pair, true))
}

func (h *Handler) refresh(c *gin.Context) {
	var body refreshRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลไม่ถูกต้อง")
		return
	}

	pair, err := h.Service.Refresh(c.Request.Context(), body.RefreshToken, c.Request.UserAgent())
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toAuthResponse(*pair, true))
}

func (h *Handler) logout(c *gin.Context) {
	var body refreshRequest
	_ = c.ShouldBindJSON(&body)

	if err := h.Service.Logout(c.Request.Context(), body.RefreshToken); err != nil {
		httpx.Internal(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) me(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	user, err := h.Service.Me(c.Request.Context(), p.UserID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toUserResponse(*user))
}

func (h *Handler) updateMe(c *gin.Context) {
	var body updateMeRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	user, err := h.Service.UpdateMe(c.Request.Context(), p.UserID, domain.UserPatch{
		DisplayName: body.DisplayName,
		AvatarURL:   body.AvatarURL,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toUserResponse(*user))
}

func (h *Handler) getPrefs(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	prefs, err := h.Service.GetPrefs(c.Request.Context(), p.UserID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toPrefsResponse(*prefs))
}

func (h *Handler) putPrefs(c *gin.Context) {
	var body prefsRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ค่าการอ่านไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	saved, err := h.Service.SetPrefs(c.Request.Context(), domain.Prefs{
		UserID:      p.UserID,
		Theme:       body.Theme,
		Font:        body.Font,
		FontSize:    body.FontSize,
		LineHeight:  body.LineHeight,
		ColumnWidth: body.ColumnWidth,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toPrefsResponse(*saved))
}

func (h *Handler) getGenrePrefs(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	prefs, err := h.Service.ListGenrePrefs(c.Request.Context(), p.UserID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toGenrePrefResponses(prefs), "")
}

func (h *Handler) putGenrePrefs(c *gin.Context) {
	var body genrePrefsRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลแนวที่ชอบไม่ถูกต้อง")
		return
	}

	prefs := make([]domain.GenrePref, 0, len(body.Genres))
	for _, g := range body.Genres {
		prefs = append(prefs, domain.GenrePref{GenreID: g.GenreID, Weight: g.Weight})
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.SetGenrePrefs(c.Request.Context(), p.UserID, prefs); err != nil {
		httpx.Internal(c, err)
		return
	}

	saved, err := h.Service.ListGenrePrefs(c.Request.Context(), p.UserID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toGenrePrefResponses(saved), "")
}

func (h *Handler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		httpx.Error(c, http.StatusConflict, "EMAIL_TAKEN", "อีเมลนี้ถูกใช้งานแล้ว")
	case errors.Is(err, domain.ErrUsernameTaken):
		httpx.Error(c, http.StatusConflict, "USERNAME_TAKEN", "ชื่อผู้ใช้นี้ถูกใช้งานแล้ว")
	case errors.Is(err, domain.ErrInvalidUsername):
		httpx.BadRequest(c, "INVALID_USERNAME", "ชื่อผู้ใช้ต้องยาว 3–32 ตัว ใช้ได้เฉพาะ a-z 0-9 . _ -")
	case errors.Is(err, domain.ErrInvalidEmail):
		httpx.BadRequest(c, "INVALID_EMAIL", "รูปแบบอีเมลไม่ถูกต้อง")
	case errors.Is(err, domain.ErrInvalidDisplayName):
		httpx.BadRequest(c, "INVALID_DISPLAY_NAME", "ชื่อที่แสดงยาวเกินไป")
	case errors.Is(err, domain.ErrWeakPassword):
		httpx.BadRequest(c, "WEAK_PASSWORD", "รหัสผ่านต้องยาวอย่างน้อย 8 ตัวอักษร")
	case errors.Is(err, domain.ErrInvalidPrefs):
		httpx.BadRequest(c, "INVALID_PREFS", "ค่าการอ่านไม่อยู่ในช่วงที่รองรับ")

	// One message for both an unknown email and a wrong password, so the
	// response never reveals whether an account exists (I-AUTH-02).
	case errors.Is(err, domain.ErrInvalidCredentials):
		httpx.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "อีเมลหรือรหัสผ่านไม่ถูกต้อง")
	case errors.Is(err, domain.ErrUserSuspended):
		httpx.Forbidden(c, "บัญชีนี้ถูกระงับการใช้งาน")
	case errors.Is(err, domain.ErrInvalidRefreshToken), errors.Is(err, domain.ErrTokenReused):
		httpx.Error(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "เซสชันหมดอายุ กรุณาเข้าสู่ระบบใหม่")
	case errors.Is(err, domain.ErrNotFound):
		httpx.NotFound(c, "ไม่พบบัญชีผู้ใช้")
	default:
		httpx.Internal(c, err)
	}
}
