package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
)

type QuizService struct {
	quizRepo    domain.QuizRepository
	attemptRepo domain.QuizAttemptRepository
	cardRepo    domain.CardRepository
	deckRepo    domain.DeckRepository
}

func NewQuizService(
	quizRepo domain.QuizRepository,
	attemptRepo domain.QuizAttemptRepository,
	cardRepo domain.CardRepository,
	deckRepo domain.DeckRepository,
) *QuizService {
	return &QuizService{
		quizRepo:    quizRepo,
		attemptRepo: attemptRepo,
		cardRepo:    cardRepo,
		deckRepo:    deckRepo,
	}
}

// Request/Response DTOs

type CreateQuizRequest struct {
	DeckID           *uuid.UUID `json:"deck_id,omitempty"`
	Title            string     `json:"title" validate:"required,min=1,max=500"`
	Description      string     `json:"description" validate:"max=2000"`
	QuizType         string     `json:"quiz_type" validate:"required,oneof=mcq true_false fill_blank mixed"`
	TimeLimitSeconds *int       `json:"time_limit_seconds,omitempty" validate:"omitempty,min=30,max=7200"`
	ShuffleQuestions *bool      `json:"shuffle_questions,omitempty"`
}

type UpdateQuizRequest struct {
	Title            *string `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description      *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	QuizType         *string `json:"quiz_type,omitempty" validate:"omitempty,oneof=mcq true_false fill_blank mixed"`
	TimeLimitSeconds *int    `json:"time_limit_seconds,omitempty" validate:"omitempty,min=30,max=7200"`
	ShuffleQuestions *bool   `json:"shuffle_questions,omitempty"`
	IsPublished      *bool   `json:"is_published,omitempty"`
}

type AddQuestionRequest struct {
	CardID        *uuid.UUID      `json:"card_id,omitempty"`
	QuestionType  string          `json:"question_type" validate:"required,oneof=mcq true_false fill_blank"`
	QuestionText  string          `json:"question_text" validate:"required,min=1"`
	Options       json.RawMessage `json:"options,omitempty"`
	CorrectAnswer string          `json:"correct_answer" validate:"required,min=1"`
	Explanation   string          `json:"explanation" validate:"max=2000"`
	Points        *int            `json:"points,omitempty" validate:"omitempty,min=1,max=100"`
}

type UpdateQuestionRequest struct {
	QuestionType  *string          `json:"question_type,omitempty" validate:"omitempty,oneof=mcq true_false fill_blank"`
	QuestionText  *string          `json:"question_text,omitempty" validate:"omitempty,min=1"`
	Options       *json.RawMessage `json:"options,omitempty"`
	CorrectAnswer *string          `json:"correct_answer,omitempty" validate:"omitempty,min=1"`
	Explanation   *string          `json:"explanation,omitempty" validate:"omitempty,max=2000"`
	Points        *int             `json:"points,omitempty" validate:"omitempty,min=1,max=100"`
}

type BatchAddQuestionsRequest struct {
	Questions []AddQuestionRequest `json:"questions" validate:"required,min=1,max=200,dive"`
}

type GenerateFromDeckRequest struct {
	DeckID       uuid.UUID `json:"deck_id" validate:"required"`
	QuestionType string    `json:"question_type" validate:"required,oneof=mcq true_false fill_blank mixed"`
	Count        int       `json:"count" validate:"required,min=1,max=100"`
}

type SubmitAnswerRequest struct {
	QuestionID uuid.UUID `json:"question_id" validate:"required"`
	Answer     string    `json:"answer" validate:"required"`
	DurationMS int       `json:"duration_ms" validate:"min=0"`
}

type AnswerResult struct {
	Answer        domain.QuizAnswer `json:"answer"`
	IsCorrect     bool              `json:"is_correct"`
	Explanation   string            `json:"explanation"`
	CorrectAnswer string            `json:"correct_answer"`
}

type QuizDetail struct {
	Quiz      domain.Quiz           `json:"quiz"`
	Questions []domain.QuizQuestion `json:"questions"`
}

type StartAttemptResponse struct {
	Attempt   domain.QuizAttempt   `json:"attempt"`
	Questions []QuestionForAttempt `json:"questions"`
}

type QuestionForAttempt struct {
	ID           uuid.UUID       `json:"id"`
	QuizID       uuid.UUID       `json:"quiz_id"`
	QuestionType string          `json:"question_type"`
	QuestionText string          `json:"question_text"`
	Options      json.RawMessage `json:"options,omitempty"`
	Position     int             `json:"position"`
	Points       int             `json:"points"`
}

type AttemptDetail struct {
	Attempt domain.QuizAttempt  `json:"attempt"`
	Answers []QuizAnswerDetail  `json:"answers"`
}

type QuizAnswerDetail struct {
	domain.QuizAnswer
	Question domain.QuizQuestion `json:"question"`
}

// Quiz CRUD

func (s *QuizService) CreateQuiz(ctx context.Context, userID uuid.UUID, req CreateQuizRequest) (*domain.Quiz, error) {
	if req.DeckID != nil {
		deck, err := s.deckRepo.GetByID(ctx, *req.DeckID)
		if err != nil {
			return nil, err
		}
		if deck.UserID != userID {
			return nil, domain.ErrForbidden
		}
	}

	shuffle := true
	if req.ShuffleQuestions != nil {
		shuffle = *req.ShuffleQuestions
	}

	quiz := &domain.Quiz{
		UserID:           userID,
		DeckID:           req.DeckID,
		Title:            req.Title,
		Description:      req.Description,
		QuizType:         req.QuizType,
		TimeLimitSeconds: req.TimeLimitSeconds,
		ShuffleQuestions: shuffle,
	}

	if err := s.quizRepo.Create(ctx, quiz); err != nil {
		return nil, err
	}
	return quiz, nil
}

func (s *QuizService) GetQuiz(ctx context.Context, userID, quizID uuid.UUID) (*QuizDetail, error) {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if quiz.UserID != userID {
		return nil, domain.ErrForbidden
	}

	questions, err := s.quizRepo.ListQuestionsByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	return &QuizDetail{Quiz: *quiz, Questions: questions}, nil
}

func (s *QuizService) ListQuizzes(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.QuizWithCounts, int, error) {
	return s.quizRepo.ListByUserID(ctx, userID, limit, offset)
}

func (s *QuizService) UpdateQuiz(ctx context.Context, userID, quizID uuid.UUID, req UpdateQuizRequest) (*domain.Quiz, error) {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if quiz.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if req.Title != nil {
		quiz.Title = *req.Title
	}
	if req.Description != nil {
		quiz.Description = *req.Description
	}
	if req.QuizType != nil {
		quiz.QuizType = *req.QuizType
	}
	if req.TimeLimitSeconds != nil {
		quiz.TimeLimitSeconds = req.TimeLimitSeconds
	}
	if req.ShuffleQuestions != nil {
		quiz.ShuffleQuestions = *req.ShuffleQuestions
	}
	if req.IsPublished != nil {
		quiz.IsPublished = *req.IsPublished
	}

	if err := s.quizRepo.Update(ctx, quiz); err != nil {
		return nil, err
	}
	return quiz, nil
}

func (s *QuizService) DeleteQuiz(ctx context.Context, userID, quizID uuid.UUID) error {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return err
	}
	if quiz.UserID != userID {
		return domain.ErrForbidden
	}
	return s.quizRepo.Delete(ctx, quizID)
}

// Question operations

func (s *QuizService) AddQuestion(ctx context.Context, userID, quizID uuid.UUID, req AddQuestionRequest) (*domain.QuizQuestion, error) {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if quiz.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if err := s.validateQuestionOptions(req.QuestionType, req.Options, req.CorrectAnswer); err != nil {
		return nil, err
	}

	count, err := s.quizRepo.CountQuestionsByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	points := 1
	if req.Points != nil {
		points = *req.Points
	}

	q := &domain.QuizQuestion{
		QuizID:        quizID,
		CardID:        req.CardID,
		QuestionType:  req.QuestionType,
		QuestionText:  req.QuestionText,
		Options:       req.Options,
		CorrectAnswer: req.CorrectAnswer,
		Explanation:   req.Explanation,
		Position:      count,
		Points:        points,
	}

	if err := s.quizRepo.CreateQuestion(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *QuizService) BatchAddQuestions(ctx context.Context, userID, quizID uuid.UUID, req BatchAddQuestionsRequest) ([]*domain.QuizQuestion, error) {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if quiz.UserID != userID {
		return nil, domain.ErrForbidden
	}

	count, err := s.quizRepo.CountQuestionsByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	questions := make([]*domain.QuizQuestion, len(req.Questions))
	for i, r := range req.Questions {
		if err := s.validateQuestionOptions(r.QuestionType, r.Options, r.CorrectAnswer); err != nil {
			return nil, fmt.Errorf("question %d: %w", i+1, err)
		}

		points := 1
		if r.Points != nil {
			points = *r.Points
		}

		questions[i] = &domain.QuizQuestion{
			QuizID:        quizID,
			CardID:        r.CardID,
			QuestionType:  r.QuestionType,
			QuestionText:  r.QuestionText,
			Options:       r.Options,
			CorrectAnswer: r.CorrectAnswer,
			Explanation:   r.Explanation,
			Position:      count + i,
			Points:        points,
		}
	}

	if err := s.quizRepo.BulkCreateQuestions(ctx, questions); err != nil {
		return nil, err
	}
	return questions, nil
}

func (s *QuizService) UpdateQuestion(ctx context.Context, userID, quizID, questionID uuid.UUID, req UpdateQuestionRequest) (*domain.QuizQuestion, error) {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if quiz.UserID != userID {
		return nil, domain.ErrForbidden
	}

	question, err := s.quizRepo.GetQuestionByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if question.QuizID != quizID {
		return nil, domain.ErrQuestionNotInQuiz
	}

	if req.QuestionType != nil {
		question.QuestionType = *req.QuestionType
	}
	if req.QuestionText != nil {
		question.QuestionText = *req.QuestionText
	}
	if req.Options != nil {
		question.Options = *req.Options
	}
	if req.CorrectAnswer != nil {
		question.CorrectAnswer = *req.CorrectAnswer
	}
	if req.Explanation != nil {
		question.Explanation = *req.Explanation
	}
	if req.Points != nil {
		question.Points = *req.Points
	}

	if err := s.validateQuestionOptions(question.QuestionType, question.Options, question.CorrectAnswer); err != nil {
		return nil, err
	}

	if err := s.quizRepo.UpdateQuestion(ctx, question); err != nil {
		return nil, err
	}
	return question, nil
}

func (s *QuizService) DeleteQuestion(ctx context.Context, userID, quizID, questionID uuid.UUID) error {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return err
	}
	if quiz.UserID != userID {
		return domain.ErrForbidden
	}

	question, err := s.quizRepo.GetQuestionByID(ctx, questionID)
	if err != nil {
		return err
	}
	if question.QuizID != quizID {
		return domain.ErrQuestionNotInQuiz
	}

	return s.quizRepo.DeleteQuestion(ctx, questionID)
}

// Generate questions from deck cards

func (s *QuizService) GenerateFromDeck(ctx context.Context, userID, quizID uuid.UUID, req GenerateFromDeckRequest) ([]*domain.QuizQuestion, error) {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if quiz.UserID != userID {
		return nil, domain.ErrForbidden
	}

	deck, err := s.deckRepo.GetByID(ctx, req.DeckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	cards, _, err := s.cardRepo.ListByDeckID(ctx, req.DeckID, domain.CardFilter{}, 10000, 0)
	if err != nil {
		return nil, err
	}

	if len(cards) < 2 {
		return nil, domain.ErrInsufficientCards
	}
	if (req.QuestionType == domain.QuestionTypeMCQ || req.QuestionType == domain.QuizTypeMixed) && len(cards) < 4 {
		return nil, domain.ErrInsufficientCards
	}

	count, err := s.quizRepo.CountQuestionsByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	// Shuffle cards and limit to requested count
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })

	numToGenerate := req.Count
	if numToGenerate > len(cards) {
		numToGenerate = len(cards)
	}

	questions := make([]*domain.QuizQuestion, 0, numToGenerate)

	for i := 0; i < numToGenerate; i++ {
		card := cards[i]
		cardID := card.ID

		qType := req.QuestionType
		if qType == domain.QuizTypeMixed {
			types := []string{domain.QuestionTypeMCQ, domain.QuestionTypeTrueFalse, domain.QuestionTypeFillBlank}
			qType = types[rng.Intn(len(types))]
		}

		var q *domain.QuizQuestion
		switch qType {
		case domain.QuestionTypeMCQ:
			q = s.generateMCQ(rng, card, cards, i)
		case domain.QuestionTypeTrueFalse:
			q = s.generateTrueFalse(rng, card, cards)
		case domain.QuestionTypeFillBlank:
			q = s.generateFillBlank(card)
		}

		q.QuizID = quizID
		q.CardID = &cardID
		q.Position = count + i
		questions = append(questions, q)
	}

	if err := s.quizRepo.BulkCreateQuestions(ctx, questions); err != nil {
		return nil, err
	}

	return questions, nil
}

func (s *QuizService) generateMCQ(rng *rand.Rand, card domain.Card, allCards []domain.Card, cardIdx int) *domain.QuizQuestion {
	// Collect distractors from other cards
	var distractors []string
	for j, c := range allCards {
		if j != cardIdx && c.Back != card.Back {
			distractors = append(distractors, c.Back)
		}
	}
	rng.Shuffle(len(distractors), func(i, j int) { distractors[i], distractors[j] = distractors[j], distractors[i] })
	if len(distractors) > 3 {
		distractors = distractors[:3]
	}

	// Build options
	opts := make([]domain.MCQOption, 0, len(distractors)+1)
	opts = append(opts, domain.MCQOption{Text: card.Back, IsCorrect: true})
	for _, d := range distractors {
		opts = append(opts, domain.MCQOption{Text: d, IsCorrect: false})
	}
	rng.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })

	optionsJSON, _ := json.Marshal(opts)

	return &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeMCQ,
		QuestionText:  card.Front,
		Options:       optionsJSON,
		CorrectAnswer: card.Back,
		Points:        1,
	}
}

func (s *QuizService) generateTrueFalse(rng *rand.Rand, card domain.Card, allCards []domain.Card) *domain.QuizQuestion {
	isTrue := rng.Intn(2) == 0

	var questionText, correctAnswer string
	if isTrue {
		questionText = fmt.Sprintf("%s: %s", card.Front, card.Back)
		correctAnswer = "true"
	} else {
		// Pick a random wrong answer
		var wrongAnswer string
		for _, c := range allCards {
			if c.Back != card.Back {
				wrongAnswer = c.Back
				break
			}
		}
		if wrongAnswer == "" {
			// Fallback if all cards have the same back
			questionText = fmt.Sprintf("%s: %s", card.Front, card.Back)
			correctAnswer = "true"
		} else {
			questionText = fmt.Sprintf("%s: %s", card.Front, wrongAnswer)
			correctAnswer = "false"
		}
	}

	return &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeTrueFalse,
		QuestionText:  questionText,
		CorrectAnswer: correctAnswer,
		Points:        1,
	}
}

func (s *QuizService) generateFillBlank(card domain.Card) *domain.QuizQuestion {
	questionText := fmt.Sprintf("%s: _____", card.Front)

	return &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeFillBlank,
		QuestionText:  questionText,
		CorrectAnswer: card.Back,
		Points:        1,
	}
}

// Attempt operations

func (s *QuizService) StartAttempt(ctx context.Context, userID, quizID uuid.UUID) (*StartAttemptResponse, error) {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	// Allow owner or published quizzes
	if quiz.UserID != userID && !quiz.IsPublished {
		return nil, domain.ErrQuizNotPublished
	}

	questions, err := s.quizRepo.ListQuestionsByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	if quiz.ShuffleQuestions {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		rng.Shuffle(len(questions), func(i, j int) { questions[i], questions[j] = questions[j], questions[i] })
	}

	// Calculate totals
	totalPoints := 0
	for _, q := range questions {
		totalPoints += q.Points
	}

	attempt := &domain.QuizAttempt{
		QuizID:         quizID,
		UserID:         userID,
		TotalPoints:    totalPoints,
		TotalQuestions: len(questions),
	}
	if err := s.attemptRepo.Create(ctx, attempt); err != nil {
		return nil, err
	}

	// Strip correct answers for attempt response
	stripped := make([]QuestionForAttempt, len(questions))
	for i, q := range questions {
		opts := q.Options
		if q.QuestionType == domain.QuestionTypeMCQ && len(opts) > 0 {
			// Remove is_correct from MCQ options
			var mcqOpts []domain.MCQOption
			if err := json.Unmarshal(opts, &mcqOpts); err == nil {
				type safeOption struct {
					Text string `json:"text"`
				}
				safe := make([]safeOption, len(mcqOpts))
				for j, o := range mcqOpts {
					safe[j] = safeOption{Text: o.Text}
				}
				opts, _ = json.Marshal(safe)
			}
		}

		stripped[i] = QuestionForAttempt{
			ID:           q.ID,
			QuizID:       q.QuizID,
			QuestionType: q.QuestionType,
			QuestionText: q.QuestionText,
			Options:      opts,
			Position:     i,
			Points:       q.Points,
		}
	}

	return &StartAttemptResponse{
		Attempt:   *attempt,
		Questions: stripped,
	}, nil
}

func (s *QuizService) SubmitAnswer(ctx context.Context, userID, attemptID uuid.UUID, req SubmitAnswerRequest) (*AnswerResult, error) {
	attempt, err := s.attemptRepo.GetByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if attempt.CompletedAt != nil {
		return nil, domain.ErrAttemptCompleted
	}

	question, err := s.quizRepo.GetQuestionByID(ctx, req.QuestionID)
	if err != nil {
		return nil, err
	}
	if question.QuizID != attempt.QuizID {
		return nil, domain.ErrQuestionNotInQuiz
	}

	// Check for duplicate answer
	existing, err := s.attemptRepo.GetAnswerByAttemptAndQuestion(ctx, attemptID, req.QuestionID)
	if err == nil && existing != nil {
		return nil, domain.ErrAlreadyAnswered
	}

	// Grade the answer
	isCorrect := s.gradeAnswer(question, req.Answer)
	pointsEarned := 0
	if isCorrect {
		pointsEarned = question.Points
	}

	answer := &domain.QuizAnswer{
		AttemptID:    attemptID,
		QuestionID:   req.QuestionID,
		UserAnswer:   req.Answer,
		IsCorrect:    isCorrect,
		PointsEarned: pointsEarned,
		DurationMS:   req.DurationMS,
	}
	if err := s.attemptRepo.CreateAnswer(ctx, answer); err != nil {
		return nil, err
	}

	// Update attempt counters
	attempt.Score += pointsEarned
	attempt.DurationMS += req.DurationMS
	if isCorrect {
		attempt.CorrectCount++
	}
	if err := s.attemptRepo.Update(ctx, attempt); err != nil {
		return nil, err
	}

	return &AnswerResult{
		Answer:        *answer,
		IsCorrect:     isCorrect,
		Explanation:   question.Explanation,
		CorrectAnswer: question.CorrectAnswer,
	}, nil
}

func (s *QuizService) CompleteAttempt(ctx context.Context, userID, attemptID uuid.UUID) (*domain.QuizAttempt, error) {
	attempt, err := s.attemptRepo.GetByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if attempt.CompletedAt != nil {
		return nil, domain.ErrAttemptCompleted
	}

	now := time.Now()
	attempt.CompletedAt = &now
	attempt.DurationMS = int(now.Sub(attempt.StartedAt).Milliseconds())

	if err := s.attemptRepo.Update(ctx, attempt); err != nil {
		return nil, err
	}
	return attempt, nil
}

func (s *QuizService) GetAttempt(ctx context.Context, userID, attemptID uuid.UUID) (*AttemptDetail, error) {
	attempt, err := s.attemptRepo.GetByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.UserID != userID {
		return nil, domain.ErrForbidden
	}

	answers, err := s.attemptRepo.ListAnswersByAttemptID(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	details := make([]QuizAnswerDetail, len(answers))
	for i, a := range answers {
		question, err := s.quizRepo.GetQuestionByID(ctx, a.QuestionID)
		if err != nil {
			return nil, err
		}
		details[i] = QuizAnswerDetail{
			QuizAnswer: a,
			Question:   *question,
		}
	}

	return &AttemptDetail{
		Attempt: *attempt,
		Answers: details,
	}, nil
}

func (s *QuizService) ListAttempts(ctx context.Context, userID, quizID uuid.UUID, limit, offset int) ([]domain.QuizAttempt, int, error) {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, 0, err
	}
	if quiz.UserID != userID {
		return nil, 0, domain.ErrForbidden
	}
	return s.attemptRepo.ListByQuizID(ctx, quizID, limit, offset)
}

// Grading logic

func (s *QuizService) gradeAnswer(question *domain.QuizQuestion, answer string) bool {
	answer = strings.TrimSpace(answer)

	switch question.QuestionType {
	case domain.QuestionTypeMCQ:
		return strings.EqualFold(answer, strings.TrimSpace(question.CorrectAnswer))

	case domain.QuestionTypeTrueFalse:
		return strings.EqualFold(answer, strings.TrimSpace(question.CorrectAnswer))

	case domain.QuestionTypeFillBlank:
		if strings.EqualFold(answer, strings.TrimSpace(question.CorrectAnswer)) {
			return true
		}
		// Check alternative answers in options
		if len(question.Options) > 0 {
			var alternatives []string
			if err := json.Unmarshal(question.Options, &alternatives); err == nil {
				for _, alt := range alternatives {
					if strings.EqualFold(answer, strings.TrimSpace(alt)) {
						return true
					}
				}
			}
		}
		return false
	}

	return false
}

// Question options validation

func (s *QuizService) validateQuestionOptions(questionType string, options json.RawMessage, correctAnswer string) error {
	switch questionType {
	case domain.QuestionTypeMCQ:
		if len(options) == 0 {
			return fmt.Errorf("%w: MCQ requires options", domain.ErrInvalidInput)
		}
		var opts []domain.MCQOption
		if err := json.Unmarshal(options, &opts); err != nil {
			return fmt.Errorf("%w: invalid MCQ options format", domain.ErrInvalidInput)
		}
		if len(opts) < 2 || len(opts) > 6 {
			return fmt.Errorf("%w: MCQ must have 2-6 options", domain.ErrInvalidInput)
		}
		correctCount := 0
		for _, o := range opts {
			if o.IsCorrect {
				correctCount++
				if o.Text != correctAnswer {
					return fmt.Errorf("%w: correct option text must match correct_answer", domain.ErrInvalidInput)
				}
			}
		}
		if correctCount != 1 {
			return fmt.Errorf("%w: MCQ must have exactly one correct option", domain.ErrInvalidInput)
		}

	case domain.QuestionTypeTrueFalse:
		a := strings.ToLower(strings.TrimSpace(correctAnswer))
		if a != "true" && a != "false" {
			return fmt.Errorf("%w: true/false correct_answer must be 'true' or 'false'", domain.ErrInvalidInput)
		}

	case domain.QuestionTypeFillBlank:
		if len(options) > 0 {
			var alternatives []string
			if err := json.Unmarshal(options, &alternatives); err != nil {
				return fmt.Errorf("%w: fill_blank options must be array of strings", domain.ErrInvalidInput)
			}
		}
	}

	return nil
}
