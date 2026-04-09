package dto

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
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
