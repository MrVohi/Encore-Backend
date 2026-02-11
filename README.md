# Encore Backend (Groupie Tracker Advanced)

> REST API for artist discovery, concerts, authentication, follows, media uploads, and notification workflows.

![Version](https://img.shields.io/badge/version-1.0.0-blue)
![Status](https://img.shields.io/badge/status-in%20development-orange)

---

## Table of contents

- [About](#about)
- [Tech stack](#tech-stack)
- [Backend scope](#backend-scope)
- [API overview](#api-overview)
- [Authentication & security](#authentication--security)
- [Data model summary](#data-model-summary)
- [Architecture](#architecture)
- [Installation](#installation)
- [Configuration](#configuration)
- [Run commands](#run-commands)
- [Tests](#tests)
- [Deployment](#deployment)
- [Observability & logging](#observability--logging)
- [Roadmap (planned)](#roadmap-planned)

---

## About

This repository contains the Go backend for the Encore / Groupie Tracker Advanced project.
It exposes a REST API used by the frontend for:

- Artist catalog and media
- Albums and tracks
- Concert management
- User authentication (email/password + Google OAuth)
- Follow/follower flows
- Email notifications for newly announced concerts
- Search across artists, albums, and tracks

---

## Tech stack

### Core

- Go 1.24
- Gin (HTTP server/router)
- PostgreSQL
- pgx (SQL access for core domains)
- GORM (authentication domain)

### Storage / Integrations

- Local filesystem uploads or Cloudflare R2 (S3-compatible)
- SMTP for transactional emails
- Google OAuth2
- Google reCAPTCHA verification
- Sentry (optional)

### Tooling

- Docker / Docker Compose
- SQL migrations runner (`cmd/migrate`)

---

## Backend scope

Implemented backend modules:

- `artist`: list/detail + admin CRUD + artwork/preview upload
- `album`: list/detail + create by artist (admin)
- `track`: list/detail + create by album (admin)
- `concert`: list/detail/by-artist + create/delete (admin)
- `follow`: follow/unfollow + follower listing
- `authentification`: register/login/refresh/me/profile/password/avatar + password reset + email verification + Google OAuth + admin user management
- `search`: text search endpoint for artist/album/track labels
- `geo`: cached geo lookup + admin resolve endpoint
- `notifications`: async email notifications for followers on new concert creation

---

## API overview

Base path: `/api`

### Health

- `GET /api/health`

### Auth

- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `GET /api/auth/google`
- `GET /api/auth/google/callback`
- `POST /api/auth/forgot-password`
- `POST /api/auth/reset-password`
- `GET /api/auth/verify-email`
- `POST /api/auth/resend-verification`
- `GET /api/auth/me` (protected)
- `PUT /api/auth/profile` (protected)
- `POST /api/auth/password` (protected)
- `POST /api/auth/avatar` (protected)
- `DELETE /api/auth/avatar` (protected)
- `POST /api/auth/logout` (protected)
- `GET /api/me` and `GET /api/user` (protected aliases)

### Admin users

- `GET /api/users` (admin)
- `POST /api/users/:id/promote` (admin)
- `DELETE /api/users/:id` (admin)

### Artists

- `GET /api/artists`
- `GET /api/artists/:id`
- `POST /api/artists` (admin)
- `PUT /api/artists/:id` (admin)
- `POST /api/artists/:id/artwork` (admin)
- `POST /api/artists/:id/preview` (admin)
- `DELETE /api/artists/:id` (admin)

### Albums / Tracks

- `GET /api/albums`
- `GET /api/albums/:id`
- `GET /api/artists/:id/albums`
- `POST /api/artists/:id/albums` (admin)
- `GET /api/tracks/:id`
- `GET /api/albums/:id/tracks`
- `POST /api/albums/:id/tracks` (admin)

### Concerts

- `GET /api/concerts`
- `GET /api/concerts/:id`
- `GET /api/artists/:id/concerts`
- `POST /api/artists/:id/concerts` (admin)
- `DELETE /api/concerts/:id` (admin)

### Follow

- `POST /api/follows` (protected)
- `DELETE /api/follows/:artist_id` (protected)
- `GET /api/follows` (protected)
- `GET /api/follows/:artist_id` (protected)
- `GET /api/follows/:artist_id/followers`

### Geo / Search / Diagnostics

- `GET /api/geo?city=...&country=...`
- `POST /api/geo/resolve` (admin)
- `GET /api/search?q=...&limit=...`
- `GET /api/sentry-test`

---

## Authentication & security

- Bearer JWT auth for protected endpoints (`Authorization: Bearer <token>`)
- Access + refresh token flow
- Role-based admin guard (`AdminOnly` middleware)
- reCAPTCHA validation on registration
- Email verification + password reset token workflows
- Google OAuth login/callback flow
- CORS policy configured for frontend and local development origins
- Secrets are loaded from environment variables

---

## Data model summary

Core persisted entities include:

- `users`, `oauth_identities`, `tokens`, `refresh_tokens`
- `artists`, `media_assets`
- `albums`, `tracks`
- `concerts`, `follow`, `concert_notifications`
- `orders`, `payments`, `ticket_types`, `tickets`
- `geo_cache`, `schema_migrations`

Notes:

- The schema already contains payments/tickets related tables.
- Purchase/payment REST workflows are not yet exposed as public API handlers in this backend.

---

## Architecture

High-level package layout:

```text
cmd/
  api/            # application entrypoint
  migrate/        # SQL migration runner
internal/
  album/
  artist/
  authentification/
  concert/
  follow/
  geo/
  health/
  http/
  media/
  middleware/
  notifications/
  search/
  storage/
  track/
pkg/
  utils/          # JWT, password hashing, random token helpers
migrations/
db/
