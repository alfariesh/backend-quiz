package dto

type CreateDeckRequest struct {
	Name           string `json:"name" validate:"required,min=1,max=200"`
	Description    string `json:"description" validate:"max=2000"`
	NewCardsPerDay *int   `json:"new_cards_per_day,omitempty" validate:"omitempty,min=0,max=9999"`
}

type UpdateDeckRequest struct {
	Name           *string `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Description    *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	IsArchived     *bool   `json:"is_archived,omitempty"`
	NewCardsPerDay *int    `json:"new_cards_per_day,omitempty" validate:"omitempty,min=0,max=9999"`
	Position       *int    `json:"position,omitempty" validate:"omitempty,min=0"`
}

type ShareDeckRequest struct {
	IsPublic bool `json:"is_public"`
}

type ExportDeck struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Cards       []ExportCard `json:"cards"`
}

type ExportCard struct {
	Front string   `json:"front"`
	Back  string   `json:"back"`
	Tags  []string `json:"tags"`
}

type ImportDeckRequest struct {
	Name  string       `json:"name" validate:"required,min=1,max=200"`
	Cards []ExportCard `json:"cards" validate:"required,min=1"`
}
