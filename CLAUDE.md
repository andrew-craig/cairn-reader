# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **Service-specific guidance:**
> - [Mobile App](/apps/mobile/CLAUDE.md) - React Native/Expo mobile application
> - [Explore Service](/services/explore/CLAUDE.md) - RSS feed fetching and content recommendation
> - [Read Service](/services/read/CLAUDE.md) - Article storage and RSS feed management
> - [User Service](/services/users/CLAUDE.md) - Authentication and user management

## Project Overview

Cairn is a read-it-later application consisting of:
- **Mobile App** (React Native/Expo): iOS and Android app for reading saved articles
- **Backend Services** (Go): Microservices for content discovery, storage, and user management
  - **Explore Service**: RSS feed fetching and content recommendation
  - **Read Service**: Article storage and user-specific metadata
  - **User Service**: Authentication and account management

## Approach to work
### 1. Plan Node Default
- Enter plan mode for ANY non-trivial task (3+ steps or architectural decisions)
- If something goes sideways, STOP and re-plan immediately - don't keep pushing
- Use plan mode for verification steps, not just building
- Write detailed specs upfront to reduce ambiguity

### 2. Subagent Strategy
- Use subagents liberally to keep main context window clean
- Offload research, exploration, and parallel analysis to subagents
- For complex problems, throw more compute at it via subagents
- One task per subagent for focused execution

### 3. Self-Improvement Loop
- After ANY correction from the user: update `tasks/lessons.md` with the pattern
- Write rules for yourself that prevent the same mistake
- Ruthlessly iterate on these lessons until mistake rate drops
- Review lessons at session start for relevant project

### 4. Verification Before Done
- Never mark a task complete without proving it works
- Diff behavior between main and your changes when relevant
- Ask yourself: "Would a staff engineer approve this?"
- Run tests, check logs, demonstrate correctness

### 5. Demand Elegance (Balanced)
- For non-trivial changes: pause and ask "is there a more elegant way?"
- If a fix feels hacky: "Knowing everything I know now, implement the elegant solution"
- Skip this for simple, obvious fixes - don't over-engineer
- Challenge your own work before presenting it

### 6. Autonomous Bug Fixing
- When given a bug report: just fix it. Don't ask for hand-holding
- Point at logs, errors, failing tests - then resolve them
- Zero context switching required from the user
- Go fix failing CI tests without being told how

## Core Principles
- **Simplicity First**: Make every change as simple as possible. Impact minimal code
- **No Laziness**: Find root causes. No temporary fixes. Senior developer standards

## Running Services

See [infrastructure/docker/README.md](infrastructure/docker/README.md) for the Docker Compose setup that runs all backend services (Vault, PostgreSQL, and all microservices).

## Architecture

```
Mobile App (React Native) → REST APIs → Backend Services
                                        ├── User Service (Auth, port 8082)
                                        ├── Explore Service (RSS, ports 8080/8081)
                                        └── Read Service (Storage, ports 8083/8085)
                                               ↓
                                        PostgreSQL
```

- Each service owns its own logical database (single PostgreSQL instance)
- Services communicate only via HTTP REST APIs — **never** access another service's database
- JWT-based stateless authentication (RS256, keys managed by HashiCorp Vault)
- All user-specific endpoints require `Authorization: Bearer <token>` header

For detailed architecture, see [docs/ARCHITECTURE.md](/docs/ARCHITECTURE.md). For per-service details, see the service CLAUDE.md files linked above.

## Cross-Service Notes

**CRITICAL**: User Service and Explore Recommender require HashiCorp Vault for JWT authentication. The Docker Compose dev setup handles this automatically.

**Authentication flow**: User Service generates JWT tokens signed with RSA private key from Vault. Other services validate tokens using the RSA public key. Services extract `user_id` from JWT claims for authorization.

## Testing

Run `make test` in any service directory. See each service's CLAUDE.md for service-specific test commands, and [docs/TESTING.md](/docs/TESTING.md) for testing standards.

## Code Conventions

See [docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md) for comprehensive coding standards, architectural principles, and style guides.

## API Documentation

Each service has an OpenAPI spec and endpoint documentation in its CLAUDE.md:
- [services/explore/api/openapi.yaml](/services/explore/api/openapi.yaml)
- [services/users/api/openapi.yaml](/services/users/api/openapi.yaml)
- [services/read/api/openapi.yaml](/services/read/api/openapi.yaml)

All services use `/health/live` (liveness) and `/health/ready` (readiness) for health checks.

## Documentation

- **Architecture**: [docs/ARCHITECTURE.md](/docs/ARCHITECTURE.md)
- **Engineering Principles**: [docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md)
- **Testing**: [docs/TESTING.md](/docs/TESTING.md)
- **Deployment**: [docs/DEPLOYMENT.md](/docs/DEPLOYMENT.md)
- **Infrastructure**: [infrastructure/docker/README.md](/infrastructure/docker/README.md)

## Task Tracking

This project uses `tsk` for task management. **Always use `tsk` for ANY task-related operation** — listing, querying, creating, updating, or closing tasks. If it is not available directly, invoke using the `/task-manager` skill. Only read the file directory directly when neither of these options work.

Tasks are stored as markdown files with YAML frontmatter at `tasks/<type>_<hex>.md` (e.g. `tasks/bug_5cc8.md`). Closed tasks move to `tasks/closed/`.

### Individual Task Tracking
1. **Setup tracking**: If there is not an existing task, create one with `tsk create` or `/task-manager`
2. **Plan First**: Write plan to the task file with checkable items
3. **Verify Plan**: Check in before starting implementation
4. **Track Progress**: Mark items complete as you go
5. **Explain Changes**: High-level summary at each step
6. **Document Results**: Add review section to the task file
7. **Capture Lessons**: Update `LEARNINGS.md` after corrections



