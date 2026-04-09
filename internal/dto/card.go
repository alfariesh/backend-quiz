package dto

type CreateCardRequest struct {
	Front       string   `json:"front" validate:"required,min=1"`
	Back        string   `json:"back" validate:"required,min=1"`
	ContentType string   `json:"content_type,omitempty" validate:"omitempty,oneof=plain markdown html"`
	Tags        []string `json:"tags"`
}

type UpdateCardRequest struct {
	Front       *string  `json:"front,omitempty" validate:"omitempty,min=1"`
	Back        *string  `json:"back,omitempty" validate:"omitempty,min=1"`
	ContentType *string  `json:"content_type,omitempty" validate:"omitempty,oneof=plain markdown html"`
	Tags        []string `json:"tags,omitempty"`
}

type BatchCreateRequest struct {
	Cards []CreateCardRequest `json:"cards" validate:"required,min=1,max=500,dive"`
}

type SuspendRequest struct {
	Suspended bool `json:"suspended"`
}
