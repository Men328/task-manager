package model

import "time"

const (
	ProviderGoogle = "google"

	DefaultTimezone = "Asia/Ho_Chi_Minh"
	DefaultLocale   = "vi"
)

type Profile struct {
	ID          string
	Email       string
	DisplayName string
	AvatarURL   string
	Timezone    string
	Locale      string
	IsActive    bool
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProfileUpdate struct {
	DisplayName *string
	AvatarURL   *string
	Timezone    *string
	Locale      *string
	IsActive    *bool
}

type AuthProvider struct {
	ID             string
	ProfileID      string
	Provider       string
	ProviderUserID string
	CreatedAt      time.Time
}

type ProviderLogin struct {
	Provider       string
	ProviderUserID string
	Email          string
	DisplayName    string
	AvatarURL      string
	Timezone       string
	Locale         string
}
