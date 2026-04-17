package dto

import "github.com/google/uuid"

type RAGQueryRequest struct {
	Mode           string     `json:"mode" validate:"required,oneof=per_kitab general"`
	KitabID        *string    `json:"kitab_id,omitempty" validate:"omitempty,uuid"`
	Question       string     `json:"question" validate:"required,min=1,max=2000"`
	Style          string     `json:"style" validate:"omitempty,oneof=ringkas standard syarah"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	ModelOverride  *string    `json:"model_override,omitempty" validate:"omitempty,min=1,max=200"`
}
