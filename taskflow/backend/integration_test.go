package main_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/Sukhveer/taskflow/internal/config"
	"github.com/Sukhveer/taskflow/internal/handler"
	"github.com/Sukhveer/taskflow/internal/infrastructure"
	"github.com/Sukhveer/taskflow/internal/infrastructure/database"
	applogger "github.com/Sukhveer/taskflow/internal/infrastructure/logger"
	postgresrepo "github.com/Sukhveer/taskflow/internal/repository/postgres"
	"github.com/Sukhveer/taskflow/internal/service"
)

type testServer struct {
	server *httptest.Server
	token  string
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres password=postgres dbname=taskflow_test sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skipping integration tests — cannot open DB: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping integration tests — cannot ping DB: %v", err)
	}
	if err := database.RunMigrations(db, "./migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:    "test-secret-key-32-chars-minimum!!",
			ExpiresIn: 24 * time.Hour,
		},
	}
	logger := applogger.New("error")
	tokenSvc := infrastructure.NewJWTTokenService(cfg.JWT)
	userRepo := postgresrepo.NewUserRepository(db)
	projectRepo := postgresrepo.NewProjectRepository(db)
	taskRepo := postgresrepo.NewTaskRepository(db)
	authSvc := service.NewAuthService(userRepo, tokenSvc, logger)
	projectSvc := service.NewProjectService(projectRepo, taskRepo, logger)
	taskSvc := service.NewTaskService(taskRepo, projectRepo, logger)
	authH := handler.NewAuthHandler(authSvc, logger)
	projectH := handler.NewProjectHandler(projectSvc, logger)
	taskH := handler.NewTaskHandler(taskSvc, logger)
	router := handler.NewRouter(authH, projectH, taskH, tokenSvc, logger)

	ts := httptest.NewServer(router)
	t.Cleanup(func() { ts.Close(); db.Close() })
	return &testServer{server: ts}
}

func (ts *testServer) do(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, ts.server.URL+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if ts.token != "" {
		req.Header.Set("Authorization", "Bearer "+ts.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(v)
}

// ── Auth Tests ──────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	ts := newTestServer(t)
	email := fmt.Sprintf("user_%d@test.com", time.Now().UnixNano())
	resp := ts.do(t, http.MethodPost, "/auth/register", map[string]string{
		"name": "Test User", "email": email, "password": "securepass",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var out map[string]any
	decodeJSON(t, resp, &out)
	if out["token"] == nil {
		t.Fatal("expected token in response")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	ts := newTestServer(t)
	email := fmt.Sprintf("dupe_%d@test.com", time.Now().UnixNano())
	body := map[string]string{"name": "User", "email": email, "password": "password123"}
	ts.do(t, http.MethodPost, "/auth/register", body).Body.Close()
	resp := ts.do(t, http.MethodPost, "/auth/register", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("duplicate register expected 400, got %d", resp.StatusCode)
	}
}

func TestRegister_MissingFields(t *testing.T) {
	ts := newTestServer(t)
	resp := ts.do(t, http.MethodPost, "/auth/register", map[string]string{"email": "no-name@test.com"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLogin_Success(t *testing.T) {
	ts := newTestServer(t)
	email := fmt.Sprintf("login_%d@test.com", time.Now().UnixNano())
	ts.do(t, http.MethodPost, "/auth/register", map[string]string{
		"name": "Login User", "email": email, "password": "mypassword",
	}).Body.Close()
	resp := ts.do(t, http.MethodPost, "/auth/login", map[string]string{
		"email": email, "password": "mypassword",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out map[string]any
	decodeJSON(t, resp, &out)
	if out["token"] == nil {
		t.Fatal("expected token in response")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	ts := newTestServer(t)
	email := fmt.Sprintf("badpw_%d@test.com", time.Now().UnixNano())
	ts.do(t, http.MethodPost, "/auth/register", map[string]string{
		"name": "Pw User", "email": email, "password": "correctpassword",
	}).Body.Close()
	resp := ts.do(t, http.MethodPost, "/auth/login", map[string]string{
		"email": email, "password": "wrongpassword",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestProtectedRoute_NoToken(t *testing.T) {
	ts := newTestServer(t)
	resp := ts.do(t, http.MethodGet, "/projects/", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// ── Project Tests ───────────────────────────────────────────────────────────

func registerAndLogin(t *testing.T, ts *testServer) {
	t.Helper()
	email := fmt.Sprintf("proj_%d@test.com", time.Now().UnixNano())
	resp := ts.do(t, http.MethodPost, "/auth/register", map[string]string{
		"name": "Proj User", "email": email, "password": "password123",
	})
	var out map[string]any
	decodeJSON(t, resp, &out)
	ts.token = out["token"].(string)
}

func TestCreateProject_Success(t *testing.T) {
	ts := newTestServer(t)
	registerAndLogin(t, ts)
	resp := ts.do(t, http.MethodPost, "/projects/", map[string]string{"name": "My Project"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var proj map[string]any
	decodeJSON(t, resp, &proj)
	if proj["id"] == nil {
		t.Fatal("expected project id in response")
	}
}

func TestDeleteProject_ForbiddenForNonOwner(t *testing.T) {
	ts := newTestServer(t)
	registerAndLogin(t, ts)

	projResp := ts.do(t, http.MethodPost, "/projects/", map[string]string{"name": "Owner's Project"})
	var proj map[string]any
	decodeJSON(t, projResp, &proj)
	projectID := proj["id"].(string)

	// Register a second user and try to delete
	email2 := fmt.Sprintf("other_%d@test.com", time.Now().UnixNano())
	authResp := ts.do(t, http.MethodPost, "/auth/register", map[string]string{
		"name": "Other", "email": email2, "password": "password123",
	})
	var auth2 map[string]any
	decodeJSON(t, authResp, &auth2)
	ts.token = auth2["token"].(string)

	resp := ts.do(t, http.MethodDelete, "/projects/"+projectID+"/", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

// ── Task Tests ──────────────────────────────────────────────────────────────

func createProjectAndTask(t *testing.T, ts *testServer) (projectID, taskID string) {
	t.Helper()
	projResp := ts.do(t, http.MethodPost, "/projects/", map[string]string{"name": "Task Test Project"})
	var proj map[string]any
	decodeJSON(t, projResp, &proj)
	projectID = proj["id"].(string)

	taskResp := ts.do(t, http.MethodPost, "/projects/"+projectID+"/tasks", map[string]any{
		"title": "Test Task", "status": "todo", "priority": "high",
	})
	var task map[string]any
	decodeJSON(t, taskResp, &task)
	taskID = task["id"].(string)
	return
}

func TestCreateTask_Success(t *testing.T) {
	ts := newTestServer(t)
	registerAndLogin(t, ts)
	projResp := ts.do(t, http.MethodPost, "/projects/", map[string]string{"name": "Task Project"})
	var proj map[string]any
	decodeJSON(t, projResp, &proj)
	projectID := proj["id"].(string)

	resp := ts.do(t, http.MethodPost, "/projects/"+projectID+"/tasks", map[string]any{
		"title": "Integration Test Task", "status": "todo", "priority": "high",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var task map[string]any
	decodeJSON(t, resp, &task)
	if task["id"] == nil {
		t.Fatal("expected task id in response")
	}
}

func TestListTasks_WithStatusFilter(t *testing.T) {
	ts := newTestServer(t)
	registerAndLogin(t, ts)
	projResp := ts.do(t, http.MethodPost, "/projects/", map[string]string{"name": "Filter Project"})
	var proj map[string]any
	decodeJSON(t, projResp, &proj)
	projectID := proj["id"].(string)

	for _, s := range []string{"todo", "done"} {
		ts.do(t, http.MethodPost, "/projects/"+projectID+"/tasks", map[string]any{
			"title": "Task " + s, "status": s,
		}).Body.Close()
	}

	resp := ts.do(t, http.MethodGet, "/projects/"+projectID+"/tasks?status=todo", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out map[string]any
	decodeJSON(t, resp, &out)
	tasks := out["data"].([]any)
	for _, item := range tasks {
		task := item.(map[string]any)
		if task["status"] != "todo" {
			t.Errorf("filter broken: got status %v", task["status"])
		}
	}
}

func TestUpdateTask_StatusTransition(t *testing.T) {
	ts := newTestServer(t)
	registerAndLogin(t, ts)
	_, taskID := createProjectAndTask(t, ts)

	status := "in_progress"
	resp := ts.do(t, http.MethodPatch, "/tasks/"+taskID+"/", map[string]any{"status": &status})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var task map[string]any
	decodeJSON(t, resp, &task)
	if task["status"] != "in_progress" {
		t.Errorf("expected in_progress, got %v", task["status"])
	}
}

func TestCreateTask_InvalidStatus(t *testing.T) {
	ts := newTestServer(t)
	registerAndLogin(t, ts)
	projResp := ts.do(t, http.MethodPost, "/projects/", map[string]string{"name": "Validation Project"})
	var proj map[string]any
	decodeJSON(t, projResp, &proj)
	projectID := proj["id"].(string)

	resp := ts.do(t, http.MethodPost, "/projects/"+projectID+"/tasks", map[string]any{
		"title": "Bad Task", "status": "flying",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d", resp.StatusCode)
	}
}
