package aiauthoring

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/durablefs"
	"github.com/yottaapp/yotta/internal/runid"
)

const (
	conversationFormat     = "yotta.ai-conversation"
	conversationVersion    = 1
	maxConversationTitle   = 80
	maxConversationMessage = 512
)

var conversationScopePattern = regexp.MustCompile(`^[a-zA-Z0-9-]{1,128}$`)

type ConversationMessage struct {
	ID          string    `json:"id"`
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"createdAt"`
	ReviewID    string    `json:"reviewId,omitempty"`
	Review      *Review   `json:"review,omitempty"`
	ProblemID   string    `json:"problemId,omitempty"`
	OperationID string    `json:"operationId,omitempty"`
}

type Conversation struct {
	Format     string                `json:"format"`
	Version    int                   `json:"version"`
	ID         string                `json:"id"`
	WorkflowID string                `json:"workflowId"`
	Title      string                `json:"title"`
	CreatedAt  time.Time             `json:"createdAt"`
	UpdatedAt  time.Time             `json:"updatedAt"`
	Messages   []ConversationMessage `json:"messages"`
}

type ConversationSummary struct {
	ID           string    `json:"id"`
	WorkflowID   string    `json:"workflowId"`
	Title        string    `json:"title"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	MessageCount int       `json:"messageCount"`
}

type ConversationStore struct {
	mu   sync.RWMutex
	root string
	byID map[string]Conversation
	now  func() time.Time
}

func NewConversationStore(root string, now func() time.Time) (*ConversationStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("AI conversation store requires a root")
	}
	if now == nil {
		now = time.Now
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create AI conversation store: %w", err)
	}
	store := &ConversationStore{root: root, byID: map[string]Conversation{}, now: now}
	return store, store.load()
}

func (s *ConversationStore) Create(workflowID string) (Conversation, error) {
	if err := validateConversationScope(workflowID); err != nil {
		return Conversation{}, err
	}
	id, err := runid.New()
	if err != nil {
		return Conversation{}, err
	}
	now := s.now().UTC()
	value := Conversation{Format: conversationFormat, Version: conversationVersion, ID: id, WorkflowID: workflowID, Title: "", CreatedAt: now, UpdatedAt: now, Messages: []ConversationMessage{}}
	s.mu.Lock()
	defer s.mu.Unlock()
	var newestEmpty *Conversation
	for _, existing := range s.byID {
		if existing.WorkflowID == workflowID && len(existing.Messages) == 0 && (newestEmpty == nil || existing.UpdatedAt.After(newestEmpty.UpdatedAt)) {
			copy := existing
			newestEmpty = &copy
		}
	}
	if newestEmpty != nil {
		return cloneConversation(*newestEmpty), nil
	}
	if err := s.write(value); err != nil && !durablefs.Committed(err) {
		return Conversation{}, err
	}
	s.byID[id] = cloneConversation(value)
	return cloneConversation(value), nil
}

func (s *ConversationStore) List(workflowID string) ([]ConversationSummary, error) {
	if err := validateConversationScope(workflowID); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]ConversationSummary, 0)
	var newestEmpty *ConversationSummary
	for _, value := range s.byID {
		if value.WorkflowID != workflowID {
			continue
		}
		summary := conversationSummary(value)
		if len(value.Messages) == 0 {
			if newestEmpty == nil || summary.UpdatedAt.After(newestEmpty.UpdatedAt) {
				copy := summary
				newestEmpty = &copy
			}
			continue
		}
		items = append(items, summary)
	}
	if newestEmpty != nil {
		items = append(items, *newestEmpty)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

func (s *ConversationStore) Get(workflowID, conversationID string) (Conversation, error) {
	if err := validateConversationIdentity(workflowID, conversationID); err != nil {
		return Conversation{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.byID[conversationID]
	if !ok || value.WorkflowID != workflowID {
		return Conversation{}, ErrConversationNotFound
	}
	return cloneConversation(value), nil
}

func (s *ConversationStore) Append(workflowID, conversationID string, message ConversationMessage) (Conversation, error) {
	if err := validateConversationIdentity(workflowID, conversationID); err != nil {
		return Conversation{}, err
	}
	message.Content = strings.TrimSpace(message.Content)
	if runid.Validate(message.ID) != nil || (message.Role != "user" && message.Role != "assistant") || message.Content == "" || len(message.Content) > maxInstructionBytes {
		return Conversation{}, errors.New("invalid AI conversation message")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.byID[conversationID]
	if !ok || value.WorkflowID != workflowID {
		return Conversation{}, ErrConversationNotFound
	}
	if len(value.Messages) >= maxConversationMessage {
		return Conversation{}, ErrConversationCapacity
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = s.now().UTC()
	}
	value.Messages = append(value.Messages, cloneConversationMessage(message))
	value.UpdatedAt = message.CreatedAt.UTC()
	if message.Role == "user" && len(value.Messages) == 1 {
		value.Title = conversationTitle(message.Content)
	}
	if err := s.write(value); err != nil && !durablefs.Committed(err) {
		return Conversation{}, err
	}
	s.byID[conversationID] = cloneConversation(value)
	return cloneConversation(value), nil
}

func (s *ConversationStore) Delete(workflowID, conversationID string) error {
	if err := validateConversationIdentity(workflowID, conversationID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.byID[conversationID]
	if !ok || value.WorkflowID != workflowID {
		return ErrConversationNotFound
	}
	if err := durablefs.Remove(s.path(value)); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(s.byID, conversationID)
	return nil
}

func (s *ConversationStore) UpdateReview(review Review) error {
	if review.ReviewID == "" {
		return errors.New("AI conversation review ID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for conversationID, value := range s.byID {
		for index := range value.Messages {
			if value.Messages[index].ReviewID != review.ReviewID {
				continue
			}
			copy := cloneReview(review)
			value.Messages[index].Review = &copy
			value.UpdatedAt = s.now().UTC()
			if err := s.write(value); err != nil && !durablefs.Committed(err) {
				return err
			}
			s.byID[conversationID] = cloneConversation(value)
			return nil
		}
	}
	return ErrConversationNotFound
}

func (s *ConversationStore) load() error {
	return filepath.WalkDir(s.root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(io.LimitReader(file, 8<<20))
		decoder.DisallowUnknownFields()
		var value Conversation
		err = decoder.Decode(&value)
		closeErr := file.Close()
		if err != nil {
			return fmt.Errorf("read AI conversation %s: %w", path, err)
		}
		if closeErr != nil {
			return closeErr
		}
		if err := validateConversation(value); err != nil {
			return fmt.Errorf("validate AI conversation %s: %w", path, err)
		}
		s.byID[value.ID] = value
		return nil
	})
}

func (s *ConversationStore) write(value Conversation) error {
	if err := validateConversation(value); err != nil {
		return err
	}
	path := s.path(value)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return durablefs.WriteFile(path, content, 0o600)
}

func (s *ConversationStore) path(value Conversation) string {
	return filepath.Join(s.root, value.WorkflowID, value.ID+".json")
}

func validateConversation(value Conversation) error {
	if value.Format != conversationFormat || value.Version != conversationVersion || runid.Validate(value.ID) != nil || validateConversationScope(value.WorkflowID) != nil || (strings.TrimSpace(value.Title) == "" && len(value.Messages) != 0) || utf8.RuneCountInString(value.Title) > maxConversationTitle || value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() || len(value.Messages) > maxConversationMessage {
		return errors.New("invalid AI conversation")
	}
	for _, message := range value.Messages {
		if runid.Validate(message.ID) != nil || (message.Role != "user" && message.Role != "assistant") || strings.TrimSpace(message.Content) == "" || len(message.Content) > maxInstructionBytes {
			return errors.New("invalid AI conversation message")
		}
	}
	return nil
}

func validateConversationScope(workflowID string) error {
	if !conversationScopePattern.MatchString(workflowID) {
		return errors.New("invalid AI conversation workflow ID")
	}
	return nil
}

func validateConversationIdentity(workflowID, conversationID string) error {
	if err := validateConversationScope(workflowID); err != nil {
		return err
	}
	if runid.Validate(conversationID) != nil {
		return errors.New("invalid AI conversation ID")
	}
	return nil
}

func conversationTitle(content string) string {
	content = strings.Join(strings.Fields(content), " ")
	runes := []rune(content)
	if len(runes) > maxConversationTitle {
		runes = runes[:maxConversationTitle]
	}
	return string(runes)
}

func conversationSummary(value Conversation) ConversationSummary {
	return ConversationSummary{ID: value.ID, WorkflowID: value.WorkflowID, Title: value.Title, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt, MessageCount: len(value.Messages)}
}

func cloneConversation(value Conversation) Conversation {
	value.Messages = append([]ConversationMessage(nil), value.Messages...)
	for index := range value.Messages {
		value.Messages[index] = cloneConversationMessage(value.Messages[index])
	}
	return value
}

func cloneConversationMessage(value ConversationMessage) ConversationMessage {
	if value.Review != nil {
		copy := *value.Review
		value.Review = &copy
	}
	return value
}

var (
	ErrConversationNotFound = errors.New("AI conversation not found")
	ErrConversationCapacity = errors.New("AI conversation message capacity exhausted")
)
