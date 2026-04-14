-- Seed data for development and testing
-- Password for test user: "password123" (bcrypt cost 12)

INSERT INTO users (id, name, email, password, created_at)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'Test User',
    'test@taskflow.dev',
    '$2a$12$xYBs/Rs52Yi6JdrdniXCjuvKqeA2ul/YerogrkYIsCbK4/4GArh2.',
    NOW()
) ON CONFLICT (email) DO NOTHING;

INSERT INTO projects (id, name, description, owner_id, created_at)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'TaskFlow Demo Project',
    'A sample project to demonstrate the API',
    'a0000000-0000-0000-0000-000000000001',
    NOW()
) ON CONFLICT DO NOTHING;

INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, created_at, updated_at)
VALUES
    (
        'c0000000-0000-0000-0000-000000000001',
        'Set up CI/CD pipeline',
        'Configure GitHub Actions for automated testing and deployment',
        'done',
        'high',
        'b0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000001',
        NOW(), NOW()
    ),
    (
        'c0000000-0000-0000-0000-000000000002',
        'Write API documentation',
        'Document all endpoints using OpenAPI / Swagger spec',
        'in_progress',
        'medium',
        'b0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000001',
        NOW(), NOW()
    ),
    (
        'c0000000-0000-0000-0000-000000000003',
        'Add rate limiting',
        'Implement per-IP rate limiting on auth endpoints',
        'todo',
        'low',
        'b0000000-0000-0000-0000-000000000001',
        NULL,
        NOW(), NOW()
    )
ON CONFLICT DO NOTHING;
