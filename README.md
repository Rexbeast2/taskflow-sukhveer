# TaskFlow API
---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture Decisions](#2-architecture-decisions)
3. [Running Locally](#3-running-locally)
4. [Running Migrations](#4-running-migrations)
5. [Test Credentials](#5-test-credentials)
6. [API Reference](#6-api-reference)
7. [What I'd Do With More Time](#7-what-id-do-with-more-time)

---

## 1. Overview

TaskFlow is a JSON REST API that covers three domains: authentication (register/login with JWT), project management (CRUD with ownership semantics), and task management (status/priority lifecycle with per-task assignees). Every endpoint past `/auth/*` is protected and enforces both authentication and fine-grained authorization meaning a logged-in user can only see and mutate data they own or are assigned to.

**What it does:**

- Register and login, receive a 24-hour JWT
- Create projects; only the owner can modify or delete them
- Create tasks within projects; assign them to other users
- Filter and paginate tasks by status or assignee
- Get per-project statistics (task counts by status and by assignee)


---

## 2. Architecture Decisions

### The layered dependency model

The codebase has four distinct layers. Each layer depends only on interfaces defined in the `internal/schema` package — never on concrete implementations from another layer.

```
HTTP Request
      │
      ▼
┌─────────────────────────────────────────┐
│  Handler Layer  (internal/handler)      │  ← decode, validate, HTTP response
│                  ↓ calls               │
│  Service Layer  (internal/service)      │  ← business rules, access control
│                  ↓ calls               │
│  Repository Layer (internal/repository)│  ← SQL, no logic
│                  ↓                     │
│  PostgreSQL                            │
└─────────────────────────────────────────┘
              ↑ all depend on
        internal/schema
    (interfaces, entities, errors)
```

`internal/schema` has zero external dependencies. It imports nothing outside the standard library. Every other package imports `schema` but `schema` imports none of them. This inversion means the entire service layer can be unit-tested with in-memory mock implementations of the repository interfaces without standing up a database.


### Authorization: 404 vs 403 on read operations

Read operations (`GET /projects/:id`, `GET /projects/:id/tasks`, `GET /projects/:id/stats`) return 404 when the requesting user has no access, even if the project exists. This is a deliberate security decision, not a bug.

If these endpoints returned 403 for "project exists but you can't see it," an attacker who knows or guesses UUID formats could write a scanner: try 10,000 UUIDs, collect the ones that return 403 vs 404. From 403 responses they learn which project IDs exist in the system, which is an information disclosure vulnerability. By returning 404 in both cases, we make the system opaque to enumeration — a stranger cannot distinguish "this ID doesn't exist" from "this ID exists but isn't yours."

Write operations (`PATCH`, `DELETE`) intentionally return 403 when the user can see the project but is not the owner. The logic is: if you can see something, you deserve to know you're not allowed to change it. If you can't see it at all, you get 404.

### The access check: one SQL query, not N+1

Project access is defined as: "you are the owner, OR you have at least one task assigned to you in this project." Rather than loading all project members into Go and checking in memory, `UserHasAccess()` runs a single query:

```sql
SELECT EXISTS (
    SELECT 1 FROM projects WHERE id = $1 AND owner_id = $2
    UNION ALL
    SELECT 1 FROM tasks   WHERE project_id = $1 AND assignee_id = $2
    LIMIT 1
)
```

`UNION ALL` with `LIMIT 1` lets PostgreSQL short-circuit at the first matching row. If the owner check succeeds, the tasks table is never scanned. The `EXISTS` wrapper means PostgreSQL stops as soon as one row is found. This is O(1) on both indexed columns.

### Why chi instead of net/http or Gorilla Mux

The standard library's `net/http` ServeMux in Go 1.22 gained path parameter support, but it still requires manual parameter extraction and has no middleware composition model. chi gives clean URL parameter extraction via `chi.URLParam()`, a composable middleware stack via `r.Use()`, and route grouping with per-group middleware specifically useful for isolating the public `/auth/*` routes from the protected API routes, which apply the `Authenticate` middleware only to the protected group. The tradeoff is one external dependency. chi is well-maintained, has had no breaking changes in years, and compiles to a thin wrapper over `net/http`.

### Why golang-migrate instead of goose or Atlas

golang-migrate integrates directly with `database/sql`, runs from inside the binary without a separate CLI tool in the production container, and uses plain `.sql` files readable by any developer without needing to know Go or a migration DSL. goose requires struct-based or Go-function migrations for complex cases, which mixes migration logic with application code. Atlas is excellent for schema diffing but is a heavyweight dependency for a project of this scope. The tradeoff with golang-migrate is that it does not automatically wrap each migration in a transaction, so a multi-statement migration that partially fails requires manual cleanup acceptable here because the single migration file is a straightforward CREATE TABLE sequence with no conditional logic.

### Multi-stage Dockerfile

The Dockerfile uses two stages: a `golang:1.22-alpine` builder that compiles the binary with `CGO_ENABLED=0` (fully static, no C runtime dependency), and a `alpine:3.19` runtime that copies only the compiled binary and migration files. The result is a ~15MB image vs ~400MB for a builder image. `CGO_ENABLED=0` is required because Alpine uses musl libc rather than glibc — dynamic binaries linked against glibc crash on musl. The `-ldflags="-w -s"` strips debug symbols and the symbol table, reducing binary size by roughly 30%.

---

## 3. Running Locally

**The only requirement is Docker and Docker Compose. Nothing else needs to be installed.**

```bash

git clone https://github.com/sukhveer/taskflow.git
cd taskflow

# Copy the environment file
cp .env.example .env

# The only required change: set a JWT secret
# Generate one with:
echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env

# Start everything
docker compose up --build

# API is ready at http://localhost:8080
# Health check:
curl http://localhost:8080/health
# → {"status":"ok"}
```

**Postgres Data Validation**

Logging into Postgres Image
```
docker exec -it taskflow_postgres psql -U postgres -d taskflow
```
Data in each table
```
SELECT * FROM users;
SELECT * FROM projects;
SELECT * FROM tasks;

```

**What happens during `docker compose up`:**

1. PostgreSQL 16 container starts and waits to become healthy (pg_isready polling)
2. API container starts, `entrypoint.sh` waits for Postgres to accept connections
3. The Go binary runs, `database.RunMigrations()` applies `000001_init_schema.up.sql`
4. The HTTP server starts on `:8080`
5. A one-shot `seed` container runs `scripts/seed.sql`, inserting the test user, project, and three tasks, then exits

**Subsequent starts** (after the first build) are faster — the image is cached and only the containers start:

```bash
docker compose up
```

**To stop and clean up:**

```bash
# Stop containers, keep database volume
docker compose down

# Stop containers and delete all data (wipes the Postgres volume)
docker compose down -v
```

---

## 4. Running Migrations

Migrations run **automatically** on every server start. No manual step is required.

The auto-run behavior uses `golang-migrate` embedded in the binary. On startup it reads all `*.up.sql` files from the `./migrations` directory, compares them against the `schema_migrations` table in Postgres, and applies any that haven't been applied yet. If migrations are already at the latest version, it exits silently (`ErrNoChange` is not treated as an error).

**Migration files:**

| File | Description |
|---|---|
| `000001_init_schema.up.sql` | Creates `users`, `projects`, `tasks` tables; `task_status` and `task_priority` enums; indexes on `email`, `owner_id`, `project_id`, `assignee_id`, `status` |
| `000001_init_schema.down.sql` | Drops all tables, enums, and the uuid extension — full rollback |

**To add a new migration:**

```bash
# Name format: NNNNNN_description.up.sql + NNNNNN_description.down.sql
# Both files must exist — the migrator rejects incomplete pairs
touch migrations/000002_add_task_tags.up.sql
touch migrations/000002_add_task_tags.down.sql
```

---

## 5. Test Credentials

The seed script (`scripts/seed.sql`) inserts the following data automatically when you run `docker compose up`:

```
Email:    test@taskflow.dev
Password: password123
```

The pre-seeded project and tasks are immediately visible after logging in.

**Login and get a token:**

```bash
curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@taskflow.dev","password":"password123"}' | jq .token
```

Copy the token value. All subsequent requests use it in the `Authorization` header:

```bash
export TOKEN="<paste token here>"

curl -s http://localhost:8080/projects/ \
  -H "Authorization: Bearer $TOKEN" | jq .
```

**Pre-seeded data:**

| Type | Name | Detail |
|---|---|---|
| User | Test User | `test@taskflow.dev` / `password123` |
| Project | TaskFlow Demo Project | Owner: Test User |
| Task 1 | Set up CI/CD pipeline | status: `done`, priority: `high` |
| Task 2 | Write API documentation | status: `in_progress`, priority: `medium` |
| Task 3 | Add rate limiting | status: `todo`, priority: `low` |

### Postman

postman [collection](/TaskFlowAPI.postman_collection.json) and [env](/TaskFlow.postman_environment.json) are attached in the project.


---

## 6. API Reference

All non-auth endpoints require `Authorization: Bearer <token>` in the request header. Tokens are obtained from `/auth/register` or `/auth/login` and expire after 24 hours.

A full Postman collection and environment file are included in the repository:
- `TaskFlow.postman_collection.json` — 28 requests across 4 folders, with pre-written test assertions
- `TaskFlow.postman_environment.json` — environment variables; `token`, `project_id`, `task_id` are auto-populated by the test scripts

Import both files into Postman, select the **TaskFlow — Local** environment, run **Login — Seed User** first, and every subsequent request will be authenticated automatically.

---

### Auth endpoints

#### `POST /auth/register`

Creates a new user account. Returns a JWT token valid for 24 hours.

**Request:**
```json
{
  "name": "Sukhveer Singh",
  "email": "sukhveer@example.com",
  "password": "mypassword"
}
```

**Response `201 Created`:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "Sukhveer Singh",
    "email": "sukhveer@example.com",
    "created_at": "2025-01-15T10:30:00Z"
  }
}
```

**Failure cases:**
- `400` — missing name, invalid email format, password under 8 characters
- `400` — email already registered (`fields.email: "already taken"`)

---

#### `POST /auth/login`

Authenticates an existing user. Returns a fresh JWT token.

**Request:**
```json
{
  "email": "sukhveer@example.com",
  "password": "mypassword"
}
```

**Response `200 OK`:** same shape as register.

**Failure cases:**
- `401` — wrong email or password (same error for both — prevents user enumeration)
- `400` — missing email or password fields

---

### Project endpoints

#### `GET /projects/`

Lists all projects the authenticated user either owns or has at least one assigned task in. Paginated.

**Query params:** `?page=1&limit=20` (defaults: page 1, limit 20, max limit 100)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "b0000000-0000-0000-0000-000000000001",
      "name": "TaskFlow Demo Project",
      "description": "A sample project",
      "owner_id": "a0000000-0000-0000-0000-000000000001",
      "created_at": "2025-01-15T10:30:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

---

#### `POST /projects/`

Creates a new project. The authenticated user becomes the owner.

**Request:**
```json
{
  "name": "Q2 Backend Sprint",
  "description": "Optional description"
}
```

**Response `201 Created`:** the project object.

**Failure cases:** `400` if name is missing or blank.

---

#### `GET /projects/:id/`

Returns the project and all its tasks embedded in a single response. Returns `404` if the project doesn't exist **or** if the requesting user has no access to it (see [Authorization note](#authorization-404-vs-403-on-read-operations) above).

**Response `200 OK`:**
```json
{
  "id": "b0000000-...",
  "name": "Q2 Backend Sprint",
  "owner_id": "a0000000-...",
  "created_at": "2025-01-15T10:30:00Z",
  "tasks": [
    {
      "id": "c0000000-...",
      "title": "Set up CI/CD",
      "status": "done",
      "priority": "high",
      "project_id": "b0000000-...",
      "assignee_id": "a0000000-...",
      "due_date": null,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    }
  ]
}
```

---

#### `PATCH /projects/:id/`

Updates the project name and/or description. Owner only. Non-owners who have assigned tasks get `403`. Users with no access at all get `404`.

**Request (all fields optional):**
```json
{
  "name": "Q2 Backend Sprint (revised)",
  "description": "Updated description"
}
```

**Response `200 OK`:** the updated project object.

---

#### `DELETE /projects/:id/`

Deletes the project and all its tasks. Owner only. Returns `204 No Content` on success.

---

#### `GET /projects/:id/stats`

Returns task counts grouped by status and by assignee name. Requires read access (owner or assigned member).

**Response `200 OK`:**
```json
{
  "project_id": "b0000000-...",
  "tasks_by_status": {
    "todo": 1,
    "in_progress": 1,
    "done": 1
  },
  "tasks_by_assignee": {
    "Sukhveer Singh": 2,
    "unassigned": 1
  }
}
```

---

### Task endpoints

#### `GET /projects/:id/tasks`

Lists tasks for a project. Supports optional filters and pagination. Requires read access.

**Query params:**
- `?status=todo` | `in_progress` | `done`
- `?assignee=<user-uuid>`
- `?page=1&limit=20`

Filters are additive — you can combine `?status=todo&assignee=<uuid>` to find unstarted tasks assigned to a specific person.

**Response `200 OK`:**
```json
{
  "data": [ /* array of task objects */ ],
  "total": 3,
  "page": 1,
  "limit": 20
}
```

---

#### `POST /projects/:id/tasks`

Creates a task inside the specified project. Project owner only — assignees cannot create tasks.

**Request:**
```json
{
  "title": "Implement JWT refresh",
  "description": "Add /auth/refresh endpoint",
  "status": "todo",
  "priority": "high",
  "assignee_id": "a1b2c3d4-...",
  "due_date": "2025-12-31"
}
```

- `title` — required
- `status` — optional, defaults to `todo`. Values: `todo` | `in_progress` | `done`
- `priority` — optional, defaults to `medium`. Values: `low` | `medium` | `high`
- `assignee_id` — optional UUID of any registered user
- `due_date` — optional, format `YYYY-MM-DD`

**Response `201 Created`:** the task object.

---

#### `PATCH /tasks/:id/`

Partially updates a task. All fields are optional — only include what changes. Allowed for project owner OR the task's current assignee.

**Request:**
```json
{
  "status": "in_progress",
  "priority": "medium",
  "title": "Implement JWT refresh (v2)",
  "assignee_id": "different-user-uuid",
  "due_date": "2026-01-15"
}
```

**Response `200 OK`:** the updated task object.

**Failure cases:**
- `400` — invalid status or priority value
- `403` — not the project owner and not the current assignee
- `404` — task not found or no access to parent project

---

#### `DELETE /tasks/:id/`

Deletes a task. Project owner only. Returns `204 No Content`.

---

### Error response format

All errors follow a consistent structure:

```json
{ "error": "human-readable message" }
```

Validation errors include field-level detail:

```json
{
  "error": "validation failed",
  "fields": {
    "email": "is required",
    "password": "must be at least 8 characters"
  }
}
```

**Status code semantics:**

| Code | Meaning | When used |
|---|---|---|
| `400` | Bad request | Validation failure, malformed JSON, invalid enum value |
| `401` | Unauthenticated | Missing, malformed, or expired token |
| `403` | Forbidden | Authenticated but not authorized for this specific action |
| `404` | Not found | Resource doesn't exist, OR exists but requester has no access |
| `500` | Server error | Unhandled internal error — logs contain details |


## 7. What I'd Do With More Time

---

### High priority — correctness and safety gaps

**Atomic project deletion.** The `Delete` operation in `project_service.go` runs two queries: `DeleteByProjectID` followed by `Delete`. These are not wrapped in a database transaction. If the process crashes between the two calls, you have orphaned tasks with no parent project and no foreign key constraint to catch them (because the project row was already deleted, or not yet deleted depending on which call failed). The fix is a `BeginTx` method on the repository or a `UnitOfWork` abstraction that passes `*sql.Tx` down through the call chain. I deliberately simplified this to keep the interface clean, but it is a real data consistency bug under failure conditions.

**Rate limiting on auth endpoints.** There is no rate limiting on `POST /auth/register` or `POST /auth/login`. Without it, both endpoints are vulnerable to credential stuffing (automated login attempts with known email/password pairs from breached datasets) and account enumeration via timing attacks. The correct fix is IP-based rate limiting with exponential backoff, implemented as a chi middleware using a token bucket or sliding window algorithm. A Redis-backed counter is the production approach; an in-memory map with a mutex is acceptable for a single instance. I omitted this because it requires a decision about storage backend (Redis adds operational complexity) and the project brief didn't specify it but I want to name it explicitly.

**JWT token revocation.** There is no mechanism to invalidate a token before its 24-hour expiry. If a user's account is compromised, or an admin wants to force-logout a session, the only option is to wait for the token to expire naturally. The production fix is a Redis-backed blocklist: on logout, store the token's `jti` (JWT ID — I'd add this claim) with TTL equal to the remaining token lifetime. The `Authenticate` middleware checks the blocklist on every request. I omitted this because it requires persistent session state, which contradicts the stateless JWT model and adds infrastructure. The tradeoff was complexity vs. security — for a first version, token expiry is acceptable.

**No logout endpoint.** Related to the above — there is no `POST /auth/logout`. Without a blocklist, a logout endpoint would be cosmetic (it could tell the client to discard the token, but the token would remain valid). I excluded it rather than implement a fake logout.

**Migration race condition in multi-instance deployments.** If two API pods start simultaneously, both will attempt to run `m.Up()` against the same database. golang-migrate uses a database-level advisory lock (`pg_try_advisory_lock`) internally, so the second instance waits and sees `ErrNoChange` — this is actually safe. However, the correct production pattern is to run migrations as a separate Kubernetes `initContainer` or `Job` that completes before the API pods start, rather than baking migration into the application binary. Mixing migration and application startup couples two concerns that have different failure modes.

**Postgres Connection and Cloud native support.** The current implementation already uses exponential backoff with jitter and configurable retry limits via environment variables, which is a good baseline. I would enhance it by enforcing a fail-fast mechanism once retries are exhausted, ensuring the application does not run without a critical dependency like PostgreSQL. Additionally, I would introduce proper liveness and readiness endpoints—where liveness checks process health and readiness validates database connectivity and functional availability. This makes the system more robust, observable, and aligned with cloud-native deployment practices. 

---

### Medium priority — operational readiness

**No request ID propagation.** The `RequestLogger` middleware logs each request with method, path, and status, but doesn't generate or propagate a request ID. This means if a 500 error occurs, you know something failed, but you can't correlate the handler log line with the service log line and the repository log line for the same request. The fix is `chimiddleware.RequestID` (already imported but not threaded through the logger) plus passing the request ID via `context.WithValue` and logging it at every layer.

**Database query timeouts are not set.** Every repository method uses `ctx` (the request context), but the request context has no deadline. A slow PostgreSQL query (e.g., a missing index, a lock wait) will block the goroutine indefinitely until the client disconnects. The fix is to wrap each DB call in a `context.WithTimeout` with an appropriate deadline (e.g., 5 seconds for normal queries, 30 seconds for stats aggregation).

**Structured errors don't include request context.** When a 500 error is logged, the log line contains the error message but not the endpoint, user ID, or request ID. Debugging production 500s requires cross-referencing timestamp ranges. Proper structured logging would include `slog.Any("error", err)`, `slog.String("endpoint", r.URL.Path)`, `slog.String("user_id", userID.String())`, and `slog.String("request_id", requestID)` in every error log.

---

### Lower priority — feature and quality completeness

**Password is not validated for complexity.** The only password rule is a minimum length of 8 characters. There is no check for common passwords, character class requirements, or maximum length (bcrypt silently truncates inputs over 72 bytes, which could create security issues if a user sets a very long password and expects the full string to be the hash input).

**No soft deletes.** Deleting a project or task is permanent. In a real application you'd want either soft deletes (adding a `deleted_at` column and filtering in all queries) or an audit log. Soft deletes enable an "undo" feature and preserve referential integrity for foreign key relationships if you later add comments, attachments, or activity history to tasks.

**No input length limits.** Task titles and project names have no maximum length enforced at the application layer (only a `VARCHAR(255)` at the database layer, which returns a cryptic PostgreSQL error rather than a clean 400). I'd add explicit length validation in the handler layer with a human-readable error message.

**The seed password is a bcrypt hash baked into `seed.sql`.** This works, but the hash was generated offline and pasted in. If the bcrypt cost is ever changed in the application, the seed hash won't reflect it. The correct approach is a seeding script written in Go that calls `bcrypt.GenerateFromPassword` at runtime, ensuring the seeded password always matches the application's current cost setting.
