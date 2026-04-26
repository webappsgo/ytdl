# Admin Panel

The admin panel is available at `/admin/` by default.

## First-Time Setup

1. Start the server
2. Copy the setup token from the console output
3. Navigate to `/admin/server/setup`
4. Enter the setup token and create your admin account
5. Save the generated API token (shown only once)

## Dashboard

The dashboard shows:

- Total downloads count
- Active downloads
- Queued downloads
- Completed downloads

## Settings

All server settings are configurable via the admin panel at `/admin/server/settings`.

## Authentication

- Admin accounts are stored in `users.db` (separate from application data)
- Passwords are hashed with Argon2id
- Sessions are cookie-based (30 days default, 90 days with "remember me")
- API tokens use the `adm_` prefix and SHA-256 hashing
