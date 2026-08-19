# Telegram Phone Auth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace manual Telegram session/chat configuration with a phone verification and channel selection flow for the single-user self-hosted app.

**Architecture:** Keep the existing Go server and Flutter web UI, but add a Telegram authorization service and REST endpoints that expose a product-friendly auth state machine. Persist finished Telegram session data and selected channel in the existing config store while keeping in-progress login state transient.

**Tech Stack:** Go, net/http, SQLite repository, Flutter Web, existing Telegram package, Go tests, Flutter widget/model tests

---

### Task 1: Backend auth surface

**Files:**
- Modify: `/workspace/internal/telegram/client.go`
- Modify: `/workspace/internal/telegram/gotd_client.go`
- Create: `/workspace/internal/telegram/auth.go`
- Test: `/workspace/internal/telegram/auth_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestAuthServiceTracksPhoneCodeAndPasswordSteps(t *testing.T) {
	svc := NewAuthService(&fakeAuthBackend{})

	status := svc.Status()
	if status.Step != AuthStepDisconnected {
		t.Fatalf("status.Step = %q, want %q", status.Step, AuthStepDisconnected)
	}

	if err := svc.Start(context.Background(), "+8613800138000"); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	status = svc.Status()
	if status.Step != AuthStepCodeRequired {
		t.Fatalf("status.Step = %q, want %q", status.Step, AuthStepCodeRequired)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/telegram -run TestAuthServiceTracksPhoneCodeAndPasswordSteps -count=1`
Expected: FAIL with undefined auth service/types.

- [ ] **Step 3: Write minimal implementation**

```go
type AuthStep string

const (
	AuthStepDisconnected AuthStep = "disconnected"
	AuthStepCodeRequired AuthStep = "code_required"
	AuthStepPasswordRequired AuthStep = "password_required"
	AuthStepConnected AuthStep = "connected"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/telegram -run TestAuthServiceTracksPhoneCodeAndPasswordSteps -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add /workspace/internal/telegram/client.go /workspace/internal/telegram/gotd_client.go /workspace/internal/telegram/auth.go /workspace/internal/telegram/auth_test.go
git commit -m "feat(telegram): add auth state service"
```

### Task 2: Config and API contract

**Files:**
- Modify: `/workspace/internal/store/models.go`
- Modify: `/workspace/internal/api/router.go`
- Modify: `/workspace/internal/api/config_handler.go`
- Create: `/workspace/internal/api/telegram_auth_handler.go`
- Test: `/workspace/internal/api/router_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestTelegramAuthStatusRouteReturnsDisconnectedState(t *testing.T) {
	auth := &fakeTelegramAuthService{}
	h := NewRouter(Dependencies{
		AppPassword:   "secret",
		SessionSecret: "",
		WebDir:        t.TempDir(),
		TelegramAuth:  auth,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/telegram/auth/status", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "single-user"})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api -run TestTelegramAuthStatusRouteReturnsDisconnectedState -count=1`
Expected: FAIL because route/dependency/service do not exist.

- [ ] **Step 3: Write minimal implementation**

```go
type TelegramAuthService interface {
	Status(context.Context) telegram.AuthStatus
	Start(context.Context, string) error
	VerifyCode(context.Context, string) error
	VerifyPassword(context.Context, string) error
	Disconnect(context.Context) error
	ListChannels(context.Context) ([]telegram.Channel, error)
	SelectChannel(context.Context, int64) error
	CreateChannel(context.Context, string) (telegram.Channel, error)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api -run TestTelegramAuthStatusRouteReturnsDisconnectedState -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add /workspace/internal/store/models.go /workspace/internal/api/router.go /workspace/internal/api/config_handler.go /workspace/internal/api/telegram_auth_handler.go /workspace/internal/api/router_test.go
git commit -m "feat(api): add telegram auth endpoints"
```

### Task 3: Runtime wiring and persistence

**Files:**
- Modify: `/workspace/cmd/server/main.go`
- Modify: `/workspace/internal/store/sqlite.go`
- Test: `/workspace/cmd/server/main_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestApplyStoredConfigKeepsTelegramSessionConnectedState(t *testing.T) {
	cfg := config.Config{}
	stored := store.SystemConfig{
		TelegramSessionBlob:  "saved-session",
		TelegramTargetChatID: 42,
	}

	got := applyStoredConfig(cfg, map[string]string{}, stored)
	if got.TelegramChatID != 42 {
		t.Fatalf("got.TelegramChatID = %d, want 42", got.TelegramChatID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/server -run TestApplyStoredConfigKeepsTelegramSessionConnectedState -count=1`
Expected: FAIL when new persisted auth fields are not wired correctly.

- [ ] **Step 3: Write minimal implementation**

```go
if env["APP_TELEGRAM_CHAT_ID"] == "" && stored.TelegramTargetChatID != 0 {
	cfg.TelegramChatID = stored.TelegramTargetChatID
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/server -run TestApplyStoredConfigKeepsTelegramSessionConnectedState -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add /workspace/cmd/server/main.go /workspace/internal/store/sqlite.go /workspace/cmd/server/main_test.go
git commit -m "feat(server): wire telegram auth persistence"
```

### Task 4: Flutter auth flow

**Files:**
- Modify: `/workspace/web/lib/models.dart`
- Modify: `/workspace/web/lib/api_client.dart`
- Modify: `/workspace/web/lib/app.dart`
- Modify: `/workspace/web/lib/screens/settings_screen.dart`
- Create: `/workspace/web/lib/screens/telegram_connect_screen.dart`
- Test: `/workspace/web/test/widget_test.dart`

- [ ] **Step 1: Write the failing test**

```dart
testWidgets('settings shows connect telegram flow when disconnected', (tester) async {
  await tester.pumpWidget(const NetdiskApp());
  expect(find.text('Connect Telegram'), findsOneWidget);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `PATH=/workspace/.tools/flutter/bin:$PATH flutter test test/widget_test.dart`
Expected: FAIL because the connect flow UI does not exist.

- [ ] **Step 3: Write minimal implementation**

```dart
class TelegramAuthStatus {
  const TelegramAuthStatus({required this.step, this.connected = false});
  final String step;
  final bool connected;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `PATH=/workspace/.tools/flutter/bin:$PATH flutter test test/widget_test.dart`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add /workspace/web/lib/models.dart /workspace/web/lib/api_client.dart /workspace/web/lib/app.dart /workspace/web/lib/screens/settings_screen.dart /workspace/web/lib/screens/telegram_connect_screen.dart /workspace/web/test/widget_test.dart
git commit -m "feat(web): add telegram connect flow"
```

### Task 5: End-to-end verification

**Files:**
- Modify: `/workspace/README.md`

- [ ] **Step 1: Write the failing test**

```go
func TestTelegramAuthStatusRouteReturnsConnectedSelectionState(t *testing.T) {
	// Extend the API tests to assert connected status and target channel fields.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api -run TestTelegramAuthStatusRouteReturnsConnectedSelectionState -count=1`
Expected: FAIL until the final response contract is complete.

- [ ] **Step 3: Write minimal implementation**

```md
- Connect Telegram with phone number, verification code, and optional 2FA password.
- Select an existing storage channel or create a dedicated one from the UI.
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./... && cd /workspace/web && PATH=/workspace/.tools/flutter/bin:$PATH flutter test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add /workspace/README.md
git commit -m "docs: describe telegram phone auth flow"
```
