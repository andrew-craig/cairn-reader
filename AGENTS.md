# AGENT INSTRUCTIONS

## Cairn Reader Overview

Cairn is a read-it-later application consisting of:
- **Mobile App** (React Native/Expo): iOS and Android app for reading saved articles
- **Web App** (React/Vite): browser client for reading saved articles
- **Backend Services** (Go): Microservices for content discovery, storage, and user management
  - **Explore Service**: RSS feed fetching and content recommendation
  - **Read Service**: Article storage and user-specific metadata
  - **User Service**: Authentication and account management
  - **Email Ingest Service**: Email-to-article ingestion pipeline

### Architecture

```
Mobile App / Web App → REST APIs → Backend Services
                                   ├── User Service (Auth)
                                   ├── Explore Service (RSS)
                                   ├── Read Service (Storage)
                                   └── Email Ingest Service
                                          ↓
                                   PostgreSQL
```

- Each service owns its own logical database (single PostgreSQL instance)
- Services communicate only via HTTP REST APIs — **never** access another service's database
- JWT-based stateless authentication (RS256, keys managed by HashiCorp Vault)
- All user-specific endpoints require `Authorization: Bearer <token>` header
- Local dev ports differ from each service's container-internal port — see [docs/ARCHITECTURE.md](/docs/ARCHITECTURE.md) for the full port table.

For detailed architecture, see [docs/ARCHITECTURE.md](/docs/ARCHITECTURE.md). For per-service details, see the service CLAUDE.md files linked below.


## Documentation
Read the following documentation as necessary for each task. do not read a document unless it is needed.

### Service documentation
- [Mobile App](/apps/mobile/CLAUDE.md) - React Native/Expo mobile application
- [Web App](/apps/web/CLAUDE.md) - React/Vite browser client
- [Explore Service](/services/explore/CLAUDE.md) - RSS feed fetching and content recommendation
- [Read Service](/services/read/CLAUDE.md) - Article storage and RSS feed management
- [User Service](/services/users/CLAUDE.md) - Authentication and user management
- [Email Ingest Service](/services/read/email/CLAUDE.md) - Email-to-article ingestion


### API Documentation

Each service has an OpenAPI spec and endpoint documentation in its CLAUDE.md:
- [services/explore/api/openapi.yaml](/services/explore/api/openapi.yaml)
- [services/users/api/openapi.yaml](/services/users/api/openapi.yaml)
- [services/read/api/openapi.yaml](/services/read/api/openapi.yaml)
- [services/read/email/api/openapi.yaml](/services/read/email/api/openapi.yaml)

All services use `/health/live` (liveness) and `/health/ready` (readiness) for health checks.

### Guidance documents
- **Architecture**: [docs/ARCHITECTURE.md](/docs/ARCHITECTURE.md)
- **Engineering Principles**: [docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md)
- **Testing**: [docs/TESTING.md](/docs/TESTING.md)
- **Deployment**: [docs/DEPLOYMENT.md](/docs/DEPLOYMENT.md)
- **Infrastructure**: [infrastructure/docker/README.md](/infrastructure/docker/README.md)


## Approach to work

- **Simplicity First**: Make every change as simple as possible. Impact minimal code
- **No Laziness**: Find root causes. No temporary fixes. Senior developer standards

### 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

### 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.
- Prefer established, well-maintained libraries when they reduce overall complexity or improve reliability. Do not reimplement common functionality without a clear reason
- Lean on dependenceis already in the project before writing your own implementation or adding packages. Do not assume a library lacks a capability without checking its documentation and types.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

### 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

### 4. Long-term architecture

- Do not preserve backward compatibility. Remove obsolte paths instead of adding compatilbility layers, fallbacks or migrations.
- Keep components modular and concerns clearly separated
- Grow the system in layers. Start from the smallest version that works end to end, add each new capability on top of a product that already works. Never trade a working product for unfinished complexity.
- Make architectural decisions for the long term. Do not accept a stopgap that only works for now and is meant to be replaced later.


### 5. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.


## Tools to use

### Code Search

Use `semble search` to find code by describing what it does or naming a symbol/identifier, instead of grep:

​```bash
semble search "authentication flow" ./my-project
semble search "save_pretrained" ./my-project
semble search "save model to disk" ./my-project --top-k 10
​```

If you anticipate doing more than one search, use `semble index` to create an index.

​```bash
semble index ./my-project -o my_index
​```

You can then reuse this index later on:

​```bash
semble search "save_pretrained" --index my_index
​```

An index is not automatically updated, so if the code changes significantly, reindex. If you notice stale results while resolving searches to files, reindex.

Use `--content docs` to search documentation and prose, `--content config` for config files (yaml, toml, etc.), or `--content all` to search code, docs, and config:

​```bash
semble search "deployment guide" ./my-project --content docs
semble search "database host port" ./my-project --content config
semble search "authentication" ./my-project --content all
​```

Use `semble find-related` to discover code similar to a known location (pass `file_path` and `line` from a prior search result):

​```bash
semble find-related src/auth.py 42 ./my-project
​```

Like search, `find-related` also accepts an `--index` argument.

`path` defaults to the current directory when omitted; git URLs are accepted.

If `semble` is not on `$PATH`, use `uvx --from "semble[mcp]" semble` in its place.

#### Workflow

1. Index the repo using `semble index -o cached_index`.
2. Start with `semble search` to find relevant chunks. Pass the index to achieve results faster.
3. Use `--content docs` for documentation, `--content config` for config files, or `--content all` for everything.
4. Inspect full files only when the returned chunk does not give enough context.
5. Optionally use `semble find-related` with a promising result's `file_path` and `line` to discover related implementations.
6. Use grep only when you need exhaustive literal matches or quick confirmation of an exact string.


### Task Tracking

ALWAYS use the `chalk` CLI tool for ALL task operations.

```bash
chalk ready                          # First command when picking up work — shows unblocked tasks by priority
chalk ready --parent=epic_0c4d       # Find available work under a specific epic
chalk show <id>                      # View full task details
chalk list --status=open             # List tasks with filters
chalk update <id> --status=in_progress  # Claim a task
chalk close <id>                     # Mark done (auto-unblocks dependents)
chalk create "Title" --parent=<id>   # Create sub-task
```

If you have attempted to use `chalk` and it is not available, tasks can be read manually. Tasks are stored as markdown files with YAML frontmatter at `.chalk/tasks/<type>_<hex>.md` (e.g. `tasks/bug_5cc8.md`). Closed tasks move to `.chalk/tasks/closed/`.

#### Workflow
1. **Setup tracking**: If there is not an existing task, create one with `chalk create`
2. **Plan First**: Write plan to the task file with checkable items
3. **Verify Plan**: Check in before starting implementation
4. **Create a branch**: Put all code fixes into a new branch so they can be tracked and merged
5. **Track Progress**: Mark items complete as you go
6. **Explain Changes**: High-level summary at each step
7. **Document Results**: Add review section to the task file
8. **Capture Lessons**: Update `LEARNINGS.md` after corrections
