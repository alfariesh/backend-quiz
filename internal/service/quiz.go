package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	"github.com/rekanesiads/backend-quiz/internal/dto"
	"github.com/rekanesiads/backend-quiz/internal/port"
)

var _ port.QuizServicer = (*QuizService)(nil)

type QuizService struct {
	quizRepo    domain.QuizRepository
	attemptRepo domain.QuizAttemptRepository
	cardRepo    domain.CardRepository
	deckRepo    domain.DeckRepository
	reviewRepo  domain.ReviewRepository
	uow         domain.UnitOfWork
}

func NewQuizService(
	quizRepo domain.QuizRepository,
	attemptRepo domain.QuizAttemptRepository,
	cardRepo domain.CardRepository,
	deckRepo domain.DeckRepository,
	reviewRepo domain.ReviewRepository,
	uow domain.UnitOfWork,
) *QuizService {
	return &QuizService{
		quizRepo:    quizRepo,
		attemptRepo: attemptRepo,
		cardRepo:    cardRepo,
		deckRepo:    deckRepo,
		reviewRepo:  reviewRepo,
		uow:         uow,
	}
}

// Request/Response DTOs

type CreateQuizRequest = dto.CreateQuizRequest
type UpdateQuizRequest = dto.UpdateQuizRequest
type AddQuestionRequest = dto.AddQuestionRequest
type UpdateQuestionRequest = dto.UpdateQuestionRequest
type BatchAddQuestionsRequest = dto.BatchAddQuestionsRequest
type GenerateFromDeckRequest = dto.GenerateFromDeckRequest
type GenerateAyatQuizRequest = dto.GenerateAyatQuizRequest
type SubmitAnswerRequest = dto.SubmitAnswerRequest
type AnswerResult = dto.AnswerResult
type QuizDetail = dto.QuizDetail
type StartAttemptResponse = dto.StartAttemptResponse
type QuestionForAttempt = dto.QuestionForAttempt
type AttemptDetail = dto.AttemptDetail
type QuizAnswerDetail = dto.QuizAnswerDetail

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

// Ayat/Hadits quiz generation

type ayatCard struct {
	Card     domain.Card
	Surat    string
	AyatNum  int
}

func parseCardTags(tags []string) (surat string, ayatNum int, ok bool) {
	for _, t := range tags {
		if strings.HasPrefix(t, "surat:") {
			surat = strings.TrimPrefix(t, "surat:")
		}
		if strings.HasPrefix(t, "ayat:") {
			fmt.Sscanf(strings.TrimPrefix(t, "ayat:"), "%d", &ayatNum)
		}
	}
	ok = surat != "" && ayatNum > 0
	return
}

func filterAyatCards(cards []domain.Card) []ayatCard {
	var result []ayatCard
	for _, c := range cards {
		surat, ayatNum, ok := parseCardTags(c.Tags)
		if ok {
			result = append(result, ayatCard{Card: c, Surat: surat, AyatNum: ayatNum})
		}
	}
	return result
}

func groupBySurat(cards []ayatCard) map[string][]ayatCard {
	groups := make(map[string][]ayatCard)
	for _, c := range cards {
		groups[c.Surat] = append(groups[c.Surat], c)
	}
	// Sort each group by ayat number
	for surat := range groups {
		g := groups[surat]
		sort.Slice(g, func(i, j int) bool { return g[i].AyatNum < g[j].AyatNum })
		groups[surat] = g
	}
	return groups
}

func (s *QuizService) GenerateAyatQuiz(ctx context.Context, userID, quizID uuid.UUID, req GenerateAyatQuizRequest) ([]*domain.QuizQuestion, error) {
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

	ayatCards := filterAyatCards(cards)
	if len(ayatCards) < 2 {
		return nil, domain.ErrInsufficientCards
	}

	groups := groupBySurat(ayatCards)

	count, err := s.quizRepo.CountQuestionsByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	var questions []*domain.QuizQuestion
	generated := 0

	for generated < req.Count {
		qType := req.QuestionType
		if qType == domain.QuizTypeMixedAyat {
			types := []string{
				domain.QuestionTypeAyatCloze,
				domain.QuestionTypeAyatContinuation,
				domain.QuestionTypeSurahID,
				domain.QuestionTypeOrdering,
			}
			qType = types[rng.Intn(len(types))]
		}

		var q *domain.QuizQuestion
		switch qType {
		case domain.QuestionTypeAyatCloze:
			q = s.generateAyatCloze(rng, ayatCards)
		case domain.QuestionTypeAyatContinuation:
			q = s.generateAyatContinuation(rng, groups)
		case domain.QuestionTypeSurahID:
			q = s.generateSurahIdentification(rng, ayatCards, groups)
		case domain.QuestionTypeOrdering:
			q = s.generateOrdering(rng, groups)
		}

		if q == nil {
			// Could not generate this type, try another
			break
		}

		q.QuizID = quizID
		q.Position = count + generated
		questions = append(questions, q)
		generated++
	}

	if len(questions) == 0 {
		return nil, domain.ErrInsufficientCards
	}

	if err := s.quizRepo.BulkCreateQuestions(ctx, questions); err != nil {
		return nil, err
	}

	return questions, nil
}

func (s *QuizService) generateAyatCloze(rng *rand.Rand, ayatCards []ayatCard) *domain.QuizQuestion {
	ac := ayatCards[rng.Intn(len(ayatCards))]
	cardID := ac.Card.ID

	words := strings.Fields(ac.Card.Front)
	if len(words) < 3 {
		// Too short to cloze, use full text as fill-blank
		return &domain.QuizQuestion{
			CardID:        &cardID,
			QuestionType:  domain.QuestionTypeAyatCloze,
			QuestionText:  "_____",
			CorrectAnswer: ac.Card.Front,
			Explanation:   fmt.Sprintf("%s - Ayat %d", ac.Surat, ac.AyatNum),
			Points:        1,
		}
	}

	// Blank out 1-2 consecutive words from the middle
	blankCount := 1
	if len(words) >= 5 {
		blankCount = 1 + rng.Intn(2) // 1 or 2
	}
	startIdx := 1 + rng.Intn(len(words)-blankCount-1)

	blankedWords := strings.Join(words[startIdx:startIdx+blankCount], " ")

	// Build question text
	before := strings.Join(words[:startIdx], " ")
	after := strings.Join(words[startIdx+blankCount:], " ")
	questionText := before + " _____ " + after

	return &domain.QuizQuestion{
		CardID:        &cardID,
		QuestionType:  domain.QuestionTypeAyatCloze,
		QuestionText:  questionText,
		CorrectAnswer: blankedWords,
		Explanation:   fmt.Sprintf("%s - Ayat %d", ac.Surat, ac.AyatNum),
		Points:        1,
	}
}

func (s *QuizService) generateAyatContinuation(rng *rand.Rand, groups map[string][]ayatCard) *domain.QuizQuestion {
	// Find surat groups with consecutive ayat
	type pair struct {
		current ayatCard
		next    ayatCard
	}
	var pairs []pair

	for _, group := range groups {
		for i := 0; i < len(group)-1; i++ {
			if group[i+1].AyatNum == group[i].AyatNum+1 {
				pairs = append(pairs, pair{current: group[i], next: group[i+1]})
			}
		}
	}

	if len(pairs) == 0 {
		return nil
	}

	p := pairs[rng.Intn(len(pairs))]
	cardID := p.next.Card.ID

	return &domain.QuizQuestion{
		CardID:        &cardID,
		QuestionType:  domain.QuestionTypeAyatContinuation,
		QuestionText:  fmt.Sprintf("Lanjutkan ayat setelah:\n%s", p.current.Card.Front),
		CorrectAnswer: p.next.Card.Front,
		Explanation:   fmt.Sprintf("%s - Ayat %d", p.next.Surat, p.next.AyatNum),
		Points:        1,
	}
}

func (s *QuizService) generateSurahIdentification(rng *rand.Rand, ayatCards []ayatCard, groups map[string][]ayatCard) *domain.QuizQuestion {
	if len(groups) < 2 {
		return nil
	}

	ac := ayatCards[rng.Intn(len(ayatCards))]
	cardID := ac.Card.ID

	// Collect unique surat names for distractors
	var otherSurats []string
	for surat := range groups {
		if surat != ac.Surat {
			otherSurats = append(otherSurats, surat)
		}
	}
	rng.Shuffle(len(otherSurats), func(i, j int) { otherSurats[i], otherSurats[j] = otherSurats[j], otherSurats[i] })

	distractorCount := 3
	if len(otherSurats) < distractorCount {
		distractorCount = len(otherSurats)
	}

	opts := make([]domain.MCQOption, 0, distractorCount+1)
	opts = append(opts, domain.MCQOption{Text: ac.Surat, IsCorrect: true})
	for i := 0; i < distractorCount; i++ {
		opts = append(opts, domain.MCQOption{Text: otherSurats[i], IsCorrect: false})
	}
	rng.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })

	optionsJSON, _ := json.Marshal(opts)

	return &domain.QuizQuestion{
		CardID:        &cardID,
		QuestionType:  domain.QuestionTypeSurahID,
		QuestionText:  fmt.Sprintf("Ayat berikut berasal dari surat apa?\n%s", ac.Card.Front),
		Options:       optionsJSON,
		CorrectAnswer: ac.Surat,
		Points:        1,
	}
}

type orderingOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func (s *QuizService) generateOrdering(rng *rand.Rand, groups map[string][]ayatCard) *domain.QuizQuestion {
	// Find surats with at least 3 consecutive ayat
	type sequence struct {
		surat string
		cards []ayatCard
	}
	var sequences []sequence

	for surat, group := range groups {
		if len(group) >= 3 {
			// Find consecutive runs
			for i := 0; i <= len(group)-3; i++ {
				run := []ayatCard{group[i]}
				for j := i + 1; j < len(group); j++ {
					if group[j].AyatNum == run[len(run)-1].AyatNum+1 {
						run = append(run, group[j])
					} else {
						break
					}
					if len(run) >= 5 {
						break
					}
				}
				if len(run) >= 3 {
					sequences = append(sequences, sequence{surat: surat, cards: run})
				}
			}
		}
	}

	if len(sequences) == 0 {
		return nil
	}

	seq := sequences[rng.Intn(len(sequences))]

	// Take 3-5 ayat
	takeCount := 3
	if len(seq.cards) >= 4 {
		takeCount = 3 + rng.Intn(min(len(seq.cards)-2, 3)) // 3-5
	}
	selected := seq.cards[:takeCount]

	// Build correct order
	correctIDs := make([]string, len(selected))
	for i, c := range selected {
		correctIDs[i] = c.Card.ID.String()
	}
	correctAnswer := strings.Join(correctIDs, ",")

	// Build shuffled options
	opts := make([]orderingOption, len(selected))
	for i, c := range selected {
		opts[i] = orderingOption{ID: c.Card.ID.String(), Text: c.Card.Front}
	}
	rng.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })

	optionsJSON, _ := json.Marshal(opts)

	return &domain.QuizQuestion{
		QuestionType:  domain.QuestionTypeOrdering,
		QuestionText:  fmt.Sprintf("Urutkan ayat-ayat berikut dari %s sesuai urutan yang benar:", seq.surat),
		Options:       optionsJSON,
		CorrectAnswer: correctAnswer,
		Explanation:   fmt.Sprintf("%s - Ayat %d-%d", seq.surat, selected[0].AyatNum, selected[len(selected)-1].AyatNum),
		Points:        2,
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

	// Update attempt counters
	attempt.Score += pointsEarned
	attempt.DurationMS += req.DurationMS
	if isCorrect {
		attempt.CorrectCount++
	}

	result := &AnswerResult{
		IsCorrect:     isCorrect,
		Explanation:   question.Explanation,
		CorrectAnswer: question.CorrectAnswer,
	}

	if err := s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.attemptRepo.CreateAnswer(ctx, answer); err != nil {
			return err
		}
		if err := s.attemptRepo.Update(ctx, attempt); err != nil {
			return err
		}

		// FSRS integration: update linked card if client provided pre-computed state
		if question.CardID != nil && req.FSRSCard != nil {
			s.applyFSRSFromQuiz(ctx, userID, *question.CardID, req, result)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	result.Answer = *answer

	return result, nil
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

// FSRS integration

func (s *QuizService) applyFSRSFromQuiz(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, req SubmitAnswerRequest, result *AnswerResult) {
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil || card.IsSuspended {
		return
	}

	if err := validateFSRSState(*req.FSRSCard); err != nil {
		return
	}

	stateBefore := card.State

	// Apply client-computed FSRS state
	applyFSRSCardState(card, *req.FSRSCard)
	if err := s.cardRepo.UpdateFSRS(ctx, card); err != nil {
		return
	}

	// Determine rating: use client-provided or default based on correctness
	rating := domain.RatingGood
	if req.Rating != nil {
		rating = domain.Rating(*req.Rating)
	} else if !result.IsCorrect {
		rating = domain.RatingAgain
	}

	now := time.Now()

	var logState FSRSLogState
	if req.FSRSLog != nil {
		logState = *req.FSRSLog
	}

	reviewLog := &domain.ReviewLog{
		CardID:        card.ID,
		UserID:        userID,
		Rating:        rating,
		State:         stateBefore,
		ScheduledDays: logState.ScheduledDays,
		ElapsedDays:   logState.ElapsedDays,
		Stability:     logState.Stability,
		Difficulty:    logState.Difficulty,
		DurationMS:    req.DurationMS,
		Source:        domain.ReviewSourceQuiz,
		ReviewedAt:    now,
	}
	if err := s.reviewRepo.Create(ctx, reviewLog); err != nil {
		return
	}

	result.CardUpdated = true
	result.NextDue = &card.Due
}

// Grading logic

func (s *QuizService) gradeAnswer(question *domain.QuizQuestion, answer string) bool {
	answer = strings.TrimSpace(answer)

	switch question.QuestionType {
	case domain.QuestionTypeMCQ:
		return strings.EqualFold(answer, strings.TrimSpace(question.CorrectAnswer))

	case domain.QuestionTypeTrueFalse:
		return strings.EqualFold(answer, strings.TrimSpace(question.CorrectAnswer))

	case domain.QuestionTypeFillBlank,
		domain.QuestionTypeAyatCloze,
		domain.QuestionTypeAyatContinuation:
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

	case domain.QuestionTypeSurahID:
		return strings.EqualFold(answer, strings.TrimSpace(question.CorrectAnswer))

	case domain.QuestionTypeOrdering:
		// Compare comma-separated order exactly
		return answer == question.CorrectAnswer
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

	case domain.QuestionTypeFillBlank,
		domain.QuestionTypeAyatCloze,
		domain.QuestionTypeAyatContinuation:
		if len(options) > 0 {
			var alternatives []string
			if err := json.Unmarshal(options, &alternatives); err != nil {
				return fmt.Errorf("%w: fill_blank/ayat options must be array of strings", domain.ErrInvalidInput)
			}
		}

	case domain.QuestionTypeSurahID:
		// Same as MCQ validation
		if len(options) == 0 {
			return fmt.Errorf("%w: surah_identification requires options", domain.ErrInvalidInput)
		}
		var opts []domain.MCQOption
		if err := json.Unmarshal(options, &opts); err != nil {
			return fmt.Errorf("%w: invalid options format", domain.ErrInvalidInput)
		}
		if len(opts) < 2 || len(opts) > 6 {
			return fmt.Errorf("%w: must have 2-6 options", domain.ErrInvalidInput)
		}

	case domain.QuestionTypeOrdering:
		// Options should be array of {id, text} objects
		if len(options) == 0 {
			return fmt.Errorf("%w: ordering requires options", domain.ErrInvalidInput)
		}
	}

	return nil
}
