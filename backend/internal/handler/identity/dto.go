package identity

import (
	"time"

	domain "github.com/mokchan/webnovel-backend/internal/domain/identity"
)

// UserResponse is the public shape of an account. It never carries the password
// hash.
type UserResponse struct {
	ID          int64    `json:"id,string"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	AvatarURL   string   `json:"avatar_url,omitempty"`
	Roles       []string `json:"roles"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at,omitempty"`
}

// AuthResponse is returned by register, login and refresh.
type AuthResponse struct {
	User         UserResponse `json:"user"`
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	ExpiresAt    string       `json:"expires_at"`
}

// PrefsResponse is the reader-settings payload synced across devices.
type PrefsResponse struct {
	Theme       string  `json:"theme"`
	Font        string  `json:"font"`
	FontSize    int     `json:"font_size"`
	LineHeight  float64 `json:"line_height"`
	ColumnWidth string  `json:"column_width"`
	UpdatedAt   string  `json:"updated_at,omitempty"`
}

// GenrePrefResponse is one onboarding taste weight.
type GenrePrefResponse struct {
	GenreID int64 `json:"genre_id,string"`
	Weight  int   `json:"weight"`
}

type registerRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type updateMeRequest struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

type prefsRequest struct {
	Theme       string  `json:"theme"`
	Font        string  `json:"font"`
	FontSize    int     `json:"font_size"`
	LineHeight  float64 `json:"line_height"`
	ColumnWidth string  `json:"column_width"`
}

type genrePrefsRequest struct {
	Genres []struct {
		GenreID int64 `json:"genre_id,string"`
		Weight  int   `json:"weight"`
	} `json:"genres"`
}

func toUserResponse(u domain.User) UserResponse {
	out := UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Roles:       u.Roles,
		Status:      u.Status,
	}
	if out.Roles == nil {
		out.Roles = []string{}
	}
	if !u.CreatedAt.IsZero() {
		out.CreatedAt = u.CreatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func toAuthResponse(pair domain.TokenPair, includeRefresh bool) AuthResponse {
	out := AuthResponse{
		User:      toUserResponse(pair.User),
		Token:     pair.AccessToken,
		ExpiresAt: pair.ExpiresAt.UTC().Format(time.RFC3339),
	}
	if includeRefresh {
		out.RefreshToken = pair.RefreshToken
	}
	return out
}

func toPrefsResponse(p domain.Prefs) PrefsResponse {
	out := PrefsResponse{
		Theme:       p.Theme,
		Font:        p.Font,
		FontSize:    p.FontSize,
		LineHeight:  p.LineHeight,
		ColumnWidth: p.ColumnWidth,
	}
	if !p.UpdatedAt.IsZero() {
		out.UpdatedAt = p.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func toGenrePrefResponses(prefs []domain.GenrePref) []GenrePrefResponse {
	out := make([]GenrePrefResponse, 0, len(prefs))
	for _, p := range prefs {
		out = append(out, GenrePrefResponse{GenreID: p.GenreID, Weight: p.Weight})
	}
	return out
}
