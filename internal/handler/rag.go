package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/alfariesh/backend-quiz/internal/dto"
	"github.com/alfariesh/backend-quiz/internal/middleware"
	"github.com/alfariesh/backend-quiz/internal/port"
	"github.com/alfariesh/backend-quiz/pkg/validate"
)

type RAGHandler struct {
	svc     port.RAGServicer
	baseURL string
	client  *http.Client
}

func NewRAGHandler(svc port.RAGServicer, baseURL string, client *http.Client) *RAGHandler {
	return &RAGHandler{svc: svc, baseURL: baseURL, client: client}
}

func (h *RAGHandler) Query(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.RAGQueryRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Mode == "per_kitab" && (req.KitabID == nil || *req.KitabID == "") {
		JSONError(w, http.StatusBadRequest, "kitab_id required for per_kitab mode")
		return
	}

	result, err := h.svc.Query(r.Context(), userID, req)
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, result)
}

// QueryStream proxies SSE from the Python rag-service. It verifies conversation
// ownership (if supplied) before opening the upstream stream, injects user_id
// from the JWT, then pipes bytes straight through with a Flusher.
func (h *RAGHandler) QueryStream(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		JSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.RAGQueryRequest
	if err := DecodeJSON(r, &req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Get().Struct(req); err != nil {
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Mode == "per_kitab" && (req.KitabID == nil || *req.KitabID == "") {
		JSONError(w, http.StatusBadRequest, "kitab_id required for per_kitab mode")
		return
	}
	if req.ConversationID != nil {
		if err := h.svc.VerifyConversationOwner(r.Context(), userID, *req.ConversationID); err != nil {
			HandleError(w, err)
			return
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		JSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	buf, err := json.Marshal(upstreamStreamBody(userID, req))
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "encode failed")
		return
	}

	// No timeout on the upstream client for SSE — the request context governs cancellation.
	streamCtx := r.Context()
	upstreamReq, err := http.NewRequestWithContext(streamCtx, http.MethodPost, h.baseURL+"/query/stream", bytes.NewReader(buf))
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "build upstream request")
		return
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Accept", "text/event-stream")

	resp, err := h.client.Do(upstreamReq)
	if err != nil {
		JSONError(w, http.StatusBadGateway, fmt.Sprintf("rag-service unavailable: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		JSONError(w, resp.StatusCode, string(b))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Copy SSE chunks. Flush after every read so events reach the client promptly.
	chunk := make([]byte, 4096)
	for {
		n, rerr := resp.Body.Read(chunk)
		if n > 0 {
			if _, werr := w.Write(chunk[:n]); werr != nil {
				return
			}
			flusher.Flush()
		}
		if rerr != nil {
			return
		}
		if streamCtx.Err() != nil {
			return
		}
	}
}

// upstreamStreamBody builds the wire-format body for rag-service/query/stream.
// Mirrors service.upstreamRequest but kept in handler to avoid cross-package coupling.
type upstreamStreamPayload struct {
	Mode           string  `json:"mode"`
	KitabID        *string `json:"kitab_id,omitempty"`
	Question       string  `json:"question"`
	Style          string  `json:"style"`
	UserID         string  `json:"user_id"`
	ConversationID *string `json:"conversation_id,omitempty"`
	ModelOverride  *string `json:"model_override,omitempty"`
}

func upstreamStreamBody(userID uuid.UUID, req dto.RAGQueryRequest) upstreamStreamPayload {
	style := req.Style
	if style == "" {
		style = "standard"
	}
	p := upstreamStreamPayload{
		Mode:          req.Mode,
		KitabID:       req.KitabID,
		Question:      req.Question,
		Style:         style,
		UserID:        userID.String(),
		ModelOverride: req.ModelOverride,
	}
	if req.ConversationID != nil {
		s := req.ConversationID.String()
		p.ConversationID = &s
	}
	return p
}

