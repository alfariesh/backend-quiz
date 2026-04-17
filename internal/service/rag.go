package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/dto"
	"github.com/alfariesh/backend-quiz/internal/port"
)

var _ port.RAGServicer = (*RAGService)(nil)

type RAGService struct {
	pool          *pgxpool.Pool
	baseURL       string
	internalToken string
	dailyLimit    int
	client        *http.Client
}

func NewRAGService(pool *pgxpool.Pool, baseURL, internalToken string, timeout time.Duration, dailyLimit int) *RAGService {
	return &RAGService{
		pool:          pool,
		baseURL:       baseURL,
		internalToken: internalToken,
		dailyLimit:    dailyLimit,
		client:        &http.Client{Timeout: timeout},
	}
}

func (s *RAGService) InternalToken() string { return s.internalToken }

func (s *RAGService) VerifyConversationOwner(ctx context.Context, userID, conversationID uuid.UUID) error {
	var owner uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT user_id FROM conversations WHERE id = $1`, conversationID,
	).Scan(&owner)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.NewAppError(domain.ErrNotFound, domain.CodeNotFound, "conversation not found")
		}
		return fmt.Errorf("verify conversation: %w", err)
	}
	if owner != userID {
		return domain.NewAppError(domain.ErrForbidden, domain.CodeForbidden, "conversation belongs to another user")
	}
	return nil
}

// CheckDailyLimit counts a user's question messages in the last 24 hours
// across all their conversations. Returns ErrRAGDailyLimitReached when the
// configured cap is exceeded. A non-positive dailyLimit disables the check.
func (s *RAGService) CheckDailyLimit(ctx context.Context, userID uuid.UUID) error {
	if s.dailyLimit <= 0 {
		return nil
	}
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		WHERE c.user_id = $1 AND m.role = 'user' AND m.created_at > now() - interval '24 hours'
	`, userID).Scan(&count)
	if err != nil {
		return fmt.Errorf("count daily messages: %w", err)
	}
	if count >= s.dailyLimit {
		return domain.NewAppErrorf(
			domain.ErrRAGDailyLimitReached, domain.CodeTooManyRequests,
			"daily RAG query limit reached (%d/%d)", count, s.dailyLimit,
		)
	}
	return nil
}

// upstreamRequest is the wire format accepted by the Python rag-service.
// kept private to this package so the Go API can evolve independently.
type upstreamRequest struct {
	Mode           string  `json:"mode"`
	KitabID        *string `json:"kitab_id,omitempty"`
	Question       string  `json:"question"`
	Style          string  `json:"style"`
	UserID         string  `json:"user_id"`
	ConversationID *string `json:"conversation_id,omitempty"`
	ModelOverride  *string `json:"model_override,omitempty"`
}

func (s *RAGService) Query(ctx context.Context, userID uuid.UUID, req dto.RAGQueryRequest) (*domain.RAGQueryResult, error) {
	if err := s.CheckDailyLimit(ctx, userID); err != nil {
		return nil, err
	}
	if req.ConversationID != nil {
		if err := s.VerifyConversationOwner(ctx, userID, *req.ConversationID); err != nil {
			return nil, err
		}
	}

	style := req.Style
	if style == "" {
		style = domain.RAGStyleStandard
	}

	body := upstreamRequest{
		Mode:          req.Mode,
		KitabID:       req.KitabID,
		Question:      req.Question,
		Style:         style,
		UserID:        userID.String(),
		ModelOverride: req.ModelOverride,
	}
	if req.ConversationID != nil {
		s := req.ConversationID.String()
		body.ConversationID = &s
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal rag request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/query", bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("build rag request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if s.internalToken != "" {
		httpReq.Header.Set("X-Internal-Token", s.internalToken)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("rag-service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("rag-service status %d: %s", resp.StatusCode, string(b))
	}

	var result domain.RAGQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode rag response: %w", err)
	}
	return &result, nil
}
