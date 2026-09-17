package model

import "time"

type Profile struct {
	ID          string
	Email       string
	DisplayName string
	AvatarURL   string
	Timezone    string
	Locale      string
	IsActive    bool
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
