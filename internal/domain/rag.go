package domain

import "github.com/google/uuid"

const (
	RAGModePerKitab = "per_kitab"
	RAGModeGeneral  = "general"

	RAGStyleRingkas  = "ringkas"
	RAGStyleStandard = "standard"
	RAGStyleSyarah   = "syarah"
)

type RAGCitation struct {
	ChunkID    int64    `json:"chunk_id"`
	KitabID    string   `json:"kitab_id"`
	KitabTitle string   `json:"kitab_title"`
	TreeNodeID string   `json:"tree_node_id"`
	TreeTitle  string   `json:"tree_title"`
	StartPage  int      `json:"start_page"`
	EndPage    int      `json:"end_page"`
	Score      *float64 `json:"score,omitempty"`
}

type RAGExtractedQuote struct {
	KutipanN   int    `json:"kutipan_n"`
	TextArabic string `json:"text_arabic"`
}

type RAGQueryResult struct {
	ConversationID      *uuid.UUID          `json:"conversation_id,omitempty"`
	Answer              string              `json:"answer"`
	ExtractedQuotes     []RAGExtractedQuote `json:"extracted_quotes"`
	SuggestedQuestions  []string            `json:"suggested_questions"`
	Citations           []RAGCitation       `json:"citations"`
	RetrievalStrategy   string              `json:"retrieval_strategy"`
	RetrievalReasoning  string              `json:"retrieval_reasoning"`
	PickedTreeNodeIDs   []string            `json:"picked_tree_node_ids"`
	TokenUsage          map[string]int      `json:"token_usage"`
	LatencyMs           int                 `json:"latency_ms"`
	ModelUsed           string              `json:"model_used"`
}
