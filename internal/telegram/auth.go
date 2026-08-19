package telegram

import (
	"context"
	"errors"
	"sync"
)

var ErrAuthNotStarted = errors.New("telegram auth not started")
var ErrChannelNotFound = errors.New("telegram channel not found")

type AuthStep string

const (
	AuthStepDisconnected     AuthStep = "disconnected"
	AuthStepCodeRequired     AuthStep = "code_required"
	AuthStepPasswordRequired AuthStep = "password_required"
	AuthStepConnected        AuthStep = "connected"
)

type User struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	PhoneMasked string `json:"phone_masked"`
}

type Channel struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Selected bool   `json:"selected"`
}

type AuthStatus struct {
	Step               AuthStep `json:"step"`
	Connected          bool     `json:"connected"`
	User               User     `json:"user"`
	Phone              string   `json:"phone"`
	PhoneMasked        string   `json:"phone_masked"`
	RequestID          string   `json:"-"`
	SessionBlob        string   `json:"-"`
	SelectedChannelID  int64    `json:"selected_channel_id"`
	SelectedChannel    string   `json:"selected_channel_title"`
}

type SendCodeResult struct {
	Phone       string
	PhoneMasked string
	RequestID   string
}

type CodeAuthResult struct {
	Step        AuthStep
	User        User
	SessionBlob string
}

type PasswordAuthResult struct {
	User        User
	SessionBlob string
}

type AuthBackend interface {
	SendCode(ctx context.Context, phone string) (SendCodeResult, error)
	VerifyCode(ctx context.Context, requestID, code string) (CodeAuthResult, error)
	VerifyPassword(ctx context.Context, requestID, password string) (PasswordAuthResult, error)
	ListChannels(ctx context.Context) ([]Channel, error)
	CreateChannel(ctx context.Context, title string) (Channel, error)
}

type AuthService struct {
	mu      sync.Mutex
	backend AuthBackend
	status  AuthStatus
	channels []Channel
}

func NewAuthService(backend AuthBackend) *AuthService {
	return &AuthService{
		backend: backend,
		status: AuthStatus{
			Step: AuthStepDisconnected,
		},
	}
}

func (s *AuthService) Status(ctx context.Context) AuthStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *AuthService) Start(ctx context.Context, phone string) error {
	res, err := s.backend.SendCode(ctx, phone)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Step = AuthStepCodeRequired
	s.status.Connected = false
	s.status.User = User{}
	s.status.Phone = res.Phone
	s.status.PhoneMasked = res.PhoneMasked
	s.status.RequestID = res.RequestID
	s.status.SessionBlob = ""
	s.status.SelectedChannelID = 0
	s.status.SelectedChannel = ""
	s.channels = nil
	return nil
}

func (s *AuthService) VerifyCode(ctx context.Context, code string) error {
	s.mu.Lock()
	requestID := s.status.RequestID
	s.mu.Unlock()
	if requestID == "" {
		return ErrAuthNotStarted
	}
	res, err := s.backend.VerifyCode(ctx, requestID, code)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Step = res.Step
	if res.Step == AuthStepConnected {
		s.status.Connected = true
		s.status.User = res.User
		s.status.SessionBlob = res.SessionBlob
	}
	return nil
}

func (s *AuthService) VerifyPassword(ctx context.Context, password string) error {
	s.mu.Lock()
	requestID := s.status.RequestID
	s.mu.Unlock()
	if requestID == "" {
		return ErrAuthNotStarted
	}
	res, err := s.backend.VerifyPassword(ctx, requestID, password)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Step = AuthStepConnected
	s.status.Connected = true
	s.status.User = res.User
	s.status.SessionBlob = res.SessionBlob
	return nil
}

func (s *AuthService) Disconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = AuthStatus{Step: AuthStepDisconnected}
	return nil
}

func (s *AuthService) ListChannels(ctx context.Context) ([]Channel, error) {
	s.mu.Lock()
	selectedID := s.status.SelectedChannelID
	cached := append([]Channel(nil), s.channels...)
	s.mu.Unlock()
	channels, err := s.backend.ListChannels(ctx)
	if err != nil {
		return nil, err
	}
	merged := mergeChannels(channels, cached)
	for i := range merged {
		merged[i].Selected = merged[i].ID == selectedID
	}
	s.mu.Lock()
	s.channels = merged
	s.mu.Unlock()
	return merged, nil
}

func mergeChannels(primary, extra []Channel) []Channel {
	seen := map[int64]bool{}
	merged := make([]Channel, 0, len(primary)+len(extra))
	for _, channel := range primary {
		merged = append(merged, channel)
		seen[channel.ID] = true
	}
	for _, channel := range extra {
		if seen[channel.ID] {
			continue
		}
		merged = append(merged, channel)
		seen[channel.ID] = true
	}
	return merged
}

func (s *AuthService) SelectChannel(ctx context.Context, channelID int64) error {
	channels, err := s.ListChannels(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, channel := range channels {
		if channel.ID == channelID {
			s.status.SelectedChannelID = channel.ID
			s.status.SelectedChannel = channel.Title
			return nil
		}
	}
	return ErrChannelNotFound
}

func (s *AuthService) CreateChannel(ctx context.Context, title string) (Channel, error) {
	channel, err := s.backend.CreateChannel(ctx, title)
	if err != nil {
		return Channel{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channels = mergeChannels(s.channels, []Channel{channel})
	s.status.SelectedChannelID = channel.ID
	s.status.SelectedChannel = channel.Title
	return channel, nil
}

func (s *AuthService) Restore(sessionBlob string, user User, channelID int64, channelTitle string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sessionBlob == "" {
		s.status = AuthStatus{Step: AuthStepDisconnected}
		s.channels = nil
		return
	}
	s.status.Step = AuthStepConnected
	s.status.Connected = true
	s.status.SessionBlob = sessionBlob
	s.status.User = user
	s.status.SelectedChannelID = channelID
	s.status.SelectedChannel = channelTitle
	if channelID != 0 && channelTitle != "" {
		s.channels = mergeChannels(s.channels, []Channel{{
			ID:       channelID,
			Title:    channelTitle,
			Selected: true,
		}})
	}
}
