package telegram

import (
	"context"
	"testing"
)

func TestAuthServiceTracksPhoneCodeAndPasswordSteps(t *testing.T) {
	svc := NewAuthService(&fakeAuthBackend{})

	status := svc.Status(context.Background())
	if status.Step != AuthStepDisconnected {
		t.Fatalf("status.Step = %q, want %q", status.Step, AuthStepDisconnected)
	}

	if err := svc.Start(context.Background(), "+8613800138000"); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	status = svc.Status(context.Background())
	if status.Step != AuthStepCodeRequired {
		t.Fatalf("status.Step = %q, want %q", status.Step, AuthStepCodeRequired)
	}
}

func TestAuthServiceSupportsPasswordAndChannelSelection(t *testing.T) {
	backend := &fakeAuthBackend{
		codeResult: CodeAuthResult{
			Step: AuthStepPasswordRequired,
		},
		passwordResult: PasswordAuthResult{
			User: User{
				ID:          7,
				DisplayName: "Demo User",
				PhoneMasked: "+86***8000",
			},
			SessionBlob: "session-blob",
		},
		channels: []Channel{
			{ID: 42, Title: "Storage Channel"},
		},
	}
	svc := NewAuthService(backend)

	if err := svc.Start(context.Background(), "+8613800138000"); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := svc.VerifyCode(context.Background(), "123456"); err != nil {
		t.Fatalf("VerifyCode returned error: %v", err)
	}
	if got := svc.Status(context.Background()).Step; got != AuthStepPasswordRequired {
		t.Fatalf("status.Step = %q, want %q", got, AuthStepPasswordRequired)
	}
	if err := svc.VerifyPassword(context.Background(), "secret"); err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if got := svc.Status(context.Background()).Step; got != AuthStepConnected {
		t.Fatalf("status.Step = %q, want %q", got, AuthStepConnected)
	}
	channels, err := svc.ListChannels(context.Background())
	if err != nil {
		t.Fatalf("ListChannels returned error: %v", err)
	}
	if len(channels) != 1 || channels[0].ID != 42 {
		t.Fatalf("channels = %#v, want channel 42", channels)
	}
	if err := svc.SelectChannel(context.Background(), 42); err != nil {
		t.Fatalf("SelectChannel returned error: %v", err)
	}
	status := svc.Status(context.Background())
	if status.SelectedChannelID != 42 {
		t.Fatalf("status.SelectedChannelID = %d, want 42", status.SelectedChannelID)
	}
	if status.SessionBlob != "session-blob" {
		t.Fatalf("status.SessionBlob = %q, want %q", status.SessionBlob, "session-blob")
	}
}

type fakeAuthBackend struct {
	codeResult     CodeAuthResult
	passwordResult PasswordAuthResult
	channels       []Channel
}

func (f *fakeAuthBackend) SendCode(ctx context.Context, phone string) (SendCodeResult, error) {
	return SendCodeResult{Phone: phone, PhoneMasked: phone, RequestID: "request-1"}, nil
}

func (f *fakeAuthBackend) VerifyCode(ctx context.Context, requestID, code string) (CodeAuthResult, error) {
	if f.codeResult.Step == "" {
		return CodeAuthResult{
			Step: AuthStepConnected,
			User: User{
				ID:          1,
				DisplayName: "Demo User",
				PhoneMasked: "+8613800138000",
			},
			SessionBlob: "session-blob",
		}, nil
	}
	return f.codeResult, nil
}

func (f *fakeAuthBackend) VerifyPassword(ctx context.Context, requestID, password string) (PasswordAuthResult, error) {
	if f.passwordResult.SessionBlob == "" {
		return PasswordAuthResult{
			User: User{
				ID:          1,
				DisplayName: "Demo User",
				PhoneMasked: "+8613800138000",
			},
			SessionBlob: "session-blob",
		}, nil
	}
	return f.passwordResult, nil
}

func (f *fakeAuthBackend) ListChannels(ctx context.Context) ([]Channel, error) {
	return f.channels, nil
}

func (f *fakeAuthBackend) CreateChannel(ctx context.Context, title string) (Channel, error) {
	return Channel{ID: 99, Title: title}, nil
}

