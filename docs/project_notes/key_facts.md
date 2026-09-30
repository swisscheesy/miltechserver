# Key Facts

This file stores project constants, configuration, and frequently-needed **non-sensitive** information for the miltechserver project.

## Security Warning

**NEVER store passwords, API keys, or sensitive credentials in this file.** This file is committed to version control.

**Safe to store:** Database hostnames, ports, project identifiers, API endpoint URLs, service account emails, environment names
**Store secrets in:** `.env` files, Azure Key Vault, environment variables, CI/CD secrets

## Project Stack

**Core Technologies:**
- Language: Go
- Web Framework: Gin
- Database: PostgreSQL
- ORM/Query Builder: Jet (with model generation)
- External Storage: Azure Blob Storage
- Authentication: Firebase Auth

## Database Configuration

**PostgreSQL:**
- Query Builder: Jet
- Model Generation: Jet (auto-generated from schema)
- Connection: Via environment variables

## API Configuration

**Endpoints:**
- Local Development: `http://localhost:8080` (typical Gin default)

## Local Development

**Services:**
- API Server: Gin (default port 8080)
- Database: PostgreSQL

## Project Structure

**Key Directories:**
- `api/` - API layer (controllers, routes, services, repositories)
- `bootstrap/` - Application initialization
- `docs/` - Documentation including project notes

## Important URLs

**Documentation:**
- Gin Framework: https://gin-gonic.com/docs/
- Jet Query Builder: https://github.com/go-jet/jet
- Firebase Auth Go SDK: https://firebase.google.com/docs/auth/admin/verify-id-tokens

<!-- Add more key facts below as you discover them -->

## Generated Jet Models

- Regenerate `.gen/` with `JET_DSN="postgresql://postgres:<password>@<host>:5432/miltech_ng?sslmode=disable" go run ./tools/jetregen` (2026-09-28).
- Never use the plain `jet` CLI: it omits `json:"snake_case"` tags, and API responses marshal these models directly, so every response key would change (`shop_id` -> `ShopID`). It also wipes `.gen/` first, and most of `.gen/` is gitignored, so git cannot restore it.
- After regenerating, `git status .gen` must show no changes to the 32 tracked `user_pmcs_*` files.


