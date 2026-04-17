package dto

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8,max=72"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6,numeric"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}

type SessionDTO struct {
	ID        string `json:"id"`
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`
	CreatedAt int64  `json:"created_at"`
	ExpiresAt int64  `json:"expires_at"`
}

type UpdateProfileRequest struct {
	DisplayName      *string   `json:"display_name,omitempty" validate:"omitempty,min=1,max=100"`
	Timezone         *string   `json:"timezone,omitempty"`
	DesiredRetention *float64  `json:"desired_retention,omitempty" validate:"omitempty,min=0.7,max=0.97"`
	DailyNewLimit    *int      `json:"daily_new_limit,omitempty" validate:"omitempty,min=0,max=9999"`
	DailyReviewLimit *int      `json:"daily_review_limit,omitempty" validate:"omitempty,min=0,max=9999"`
	FSRSWeights      []float64 `json:"fsrs_weights,omitempty" validate:"omitempty,len=19"`
	ReminderEnabled  *bool     `json:"reminder_enabled,omitempty"`
	ReminderTime     *string   `json:"reminder_time,omitempty" validate:"omitempty,len=5"`
}
