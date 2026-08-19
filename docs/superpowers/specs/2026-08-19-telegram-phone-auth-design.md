# Telegram Phone Auth Design

**Goal**

Turn Telegram setup from a manual config flow into a single-user self-hosted authorization flow where the user logs in with phone number, SMS/Telegram code, and optional 2FA password, then selects or creates a storage channel.

**Context**

The current app exposes Telegram setup as technical storage settings such as `telegram_session_blob` and `telegram_target_chat_id`. That is functional for development but too technical for end users and blocks a product-like onboarding flow.

## Product behavior

- On first use, the user should connect Telegram from the UI instead of pasting technical session data.
- The auth flow must support:
  - phone number submission
  - verification code submission
  - optional Telegram 2FA password
- After successful login, the app loads the Telegram account profile and the list of candidate channels.
- The user can either:
  - choose an existing channel
  - create a new dedicated storage channel and bind it immediately
- After binding, the app persists the Telegram session and selected channel for future restarts.

## Scope

### In scope

- Single-user self-hosted mode only
- Telegram authorization state machine on the backend
- API endpoints for auth start, code verify, 2FA verify, auth status, channel listing, channel bind, and channel creation
- Frontend onboarding UI for Telegram connect and channel selection
- Persisting Telegram authorization/session state and selected target channel in the existing config store
- Startup behavior that restores persisted Telegram auth state

### Out of scope

- Multi-user account separation
- QR login
- Full account management such as logout from all devices
- Non-Telegram auth providers

## Architecture

### Backend

Add an authorization-oriented Telegram service layer on top of the existing Telegram client package. This layer owns:

- current authorization status
- pending phone login transaction state
- verification flow transitions
- account profile lookup
- channel discovery
- channel creation/binding

The existing config store remains the persistence source for:

- saved Telegram session blob
- selected target channel id

### Frontend

Replace the current Telegram technical settings experience with a guided settings flow:

- disconnected state
- phone entry
- code entry
- 2FA password entry when required
- connected state with account summary and channel selection actions

The chunk size and cache settings can remain in the settings page, but Telegram session blob input should disappear from the product UI.

## Data model changes

Extend config/status responses to expose product state instead of raw internals:

- `telegram_connected`
- `telegram_user_display_name`
- `telegram_phone_masked`
- `telegram_target_chat_id`
- `telegram_target_chat_title`
- `telegram_auth_step`

Use transient in-memory state for the active login attempt and persisted state for the finished authorization session.

## API design

Planned endpoints:

- `GET /api/telegram/auth/status`
- `POST /api/telegram/auth/start`
- `POST /api/telegram/auth/verify-code`
- `POST /api/telegram/auth/verify-password`
- `POST /api/telegram/auth/disconnect`
- `GET /api/telegram/channels`
- `POST /api/telegram/channels/select`
- `POST /api/telegram/channels/create`

All endpoints remain protected by the existing single-user app session.

## Error handling

The UI should show user-readable errors for:

- invalid phone number
- invalid or expired verification code
- wrong 2FA password
- Telegram auth session expired
- no writable/usable channels found
- channel creation failure

Server responses should keep technical detail out of the default user message while still logging enough context for debugging.

## Migration approach

- Preserve existing persisted `telegram_target_chat_id`
- Preserve existing `telegram_session_blob` as the restored connected state when present
- Hide manual session entry from the UI even if the backend still stores the session blob internally

## Testing

- Backend handler tests for auth flow states and channel actions
- Backend service tests for state transitions
- Frontend widget/model tests for auth step rendering and channel selection
- End-to-end verification via local server restart with persisted config

## Acceptance criteria

- A fresh user can connect Telegram without entering technical Telegram fields.
- The flow supports phone number, verification code, and Telegram 2FA password.
- After connect, the user can select an existing channel or create a new one.
- The chosen channel becomes the persisted storage target.
- After restart, the app restores the Telegram connected state and selected channel.
