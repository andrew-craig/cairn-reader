# URL Detection and Submission Implementation Plan

## Overview
This plan implements a user-friendly URL submission flow where the mobile app can detect if a URL is an RSS feed or a web page, and submit it accordingly. The system handles feed subscriptions automatically when a feed URL is detected.

## Key Architecture Decisions

### Service Boundaries
- **Content Service** (port 8083): Handles URL detection and routing
- **Ingest RSS Service** (port 8085): Manages feed subscriptions and user-feed relationships
- **Content Service delegates to Ingest RSS** for feed subscription operations

### Feed Subscription Management
- Ingest RSS service manages the `feed_subscriptions` table (user_id ↔ feed_id)
- Multiple users can subscribe to the same feed (feed is shared)
- Feed subscription limit (100 feeds/user) enforced by Ingest RSS service
- Duplicate subscription attempts return "already_subscribed" error (409 Conflict)

### Error Handling
User-facing error messages:
- "Already subscribed to this feed" - when user re-subscribes
- "Failed to subscribe to feed" - generic feed subscription error
- "Failed to add article" - page fetch/processing error

## User Experience Flow

```
1. User enters URL in mobile app
2. Mobile immediately calls URL detection endpoint (non-blocking, 10s timeout)
3. User can submit immediately (results in "unknown" type)
   OR wait for detection to complete
4. If detection completes:
   - UI updates to show "Add" → "Add feed" if feed detected
   - User submits with known type
5. Backend processes submission:
   - For "feed": Creates feed subscription in Ingest RSS service
   - For "page" or "unknown": Fetches content and adds to reading list
```

## API Design

### 1. URL Detection Endpoint

**Endpoint:** `POST /api/v1/content/detect`

**Request:**
```json
{
  "url": "https://example.com/feed.xml"
}
```

**Response (Success):**
```json
{
  "url": "https://example.com/feed.xml",
  "type": "feed",
  "title": "Example Blog"
}
```

**Response (Not a feed):**
```json
{
  "url": "https://example.com",
  "type": "page",
  "title": "Example Website"
}
```

**Response (Error - timeout or fetch failure):**
```json
{
  "url": "https://example.com",
  "type": "unknown",
  "title": null
}
```

**Fields:**
- `url` (string): The URL that was checked (may be different if redirected)
- `type` (string): One of: `"feed"`, `"page"`, `"unknown"`
- `title` (string|null): Feed title or page title if available

**Implementation Notes:**
- 10 second timeout for HTTP fetch
- Uses existing feed parser (`gofeed`) to detect RSS/Atom feeds
- Falls back to fetching page and extracting `<title>` for pages
- Returns `"unknown"` on timeout or error

---

### 2. URL Submission Endpoint

**Endpoint:** `POST /api/v1/content/user/{user_id}`

**Request:**
```json
{
  "url": "https://example.com/feed.xml",
  "type": "feed|page|unknown",
  "title": "Example Blog"
}
```

**Fields:**
- `url` (string, required): The URL to add
- `type` (string, optional): Type hint from detection: `"feed"`, `"page"`, or `"unknown"`. Defaults to `"unknown"`
- `title` (string, optional): Pre-detected title (optimization to avoid re-fetching)

**Response (Feed):**
```json
{
  "type": "feed",
  "feed_id": "uuid",
  "subscription": {
    "id": "uuid",
    "user_id": "uuid",
    "feed_id": "uuid",
    "feed_url": "https://example.com/feed.xml",
    "title": "Example Blog",
    "subscribed_at": "2025-01-02T10:00:00Z"
  }
}
```

**Response (Page):**
```json
{
  "type": "page",
  "content": {
    "id": "uuid",
    "user_id": "uuid",
    "content_id": "uuid",
    "status": "unread",
    "scroll_position": 0,
    "is_favorite": false,
    "added_at": "2025-01-02T10:00:00Z",
    "updated_at": "2025-01-02T10:00:00Z",
    "content": {
      "id": "uuid",
      "title": "Example Article",
      "original_url": "https://example.com/article",
      ...
    }
  }
}
```

**Processing Logic:**

1. **If `type === "feed"`:**
   - Call Ingest RSS service to create/get feed
   - Subscribe user to feed
   - Return subscription details

2. **If `type === "page"`:**
   - Fetch and process content using Content Service
   - Add to user's reading list
   - Return user content

3. **If `type === "unknown"`:**
   - Attempt detection again (10s timeout)
   - If detected as `"feed"`: Follow feed flow
   - If detected as `"page"` or timeout: Follow page flow (default)

---

## Backend Implementation

### Phase 1: URL Detection Service (Content Service)

**File:** `services/read/content/internal/service/url_detector.go`

```go
package service

type URLType string

const (
    URLTypeFeed    URLType = "feed"
    URLTypePage    URLType = "page"
    URLTypeUnknown URLType = "unknown"
)

type URLDetectionResult struct {
    URL   string   `json:"url"`
    Type  URLType  `json:"type"`
    Title *string  `json:"title"`
}

type URLDetector interface {
    DetectURL(ctx context.Context, url string) (*URLDetectionResult, error)
}
```

**Implementation Steps:**
1. Create HTTP client with 10s timeout
2. Fetch URL with context timeout
3. Detect content type from HTTP headers
4. If XML/RSS content-type or feed-like URL:
   - Try parsing with gofeed parser
   - If successful, return `"feed"` with feed title
5. If HTML or other:
   - Try extracting page title from HTML
   - Return `"page"` with page title
6. On error/timeout:
   - Return `"unknown"` with nil title

**Dependencies:**
- Import feed parser from Ingest RSS service (or vendor `gofeed` directly)
- Reuse existing HTTP client configuration
- Add HTML parser for extracting `<title>` tag

---

### Phase 2: Enhanced User Content Handler

**File:** `services/read/content/internal/api/handlers/user_content_handler.go`

**New Method:** `AddContentToUserFromURL`

**Processing Steps:**

1. **Parse request:**
   ```go
   type AddContentToUserFromURLRequest struct {
       URL   string  `json:"url"`
       Type  *string `json:"type,omitempty"`  // "feed", "page", "unknown"
       Title *string `json:"title,omitempty"` // Pre-detected title
   }
   ```

2. **Determine type:**
   - If `Type` is nil or `"unknown"`: Run detection
   - Otherwise: Use provided type

3. **Route based on type:**

   **For Feed:**
   ```go
   // Call Ingest RSS service API
   POST /api/v1/source/rss/user/{user_id}/subscription
   Body: {"feed_url": url}

   // Ingest RSS service handles:
   // 1. Check if user already subscribed → 409 "already_subscribed" error
   // 2. Check if feed exists by URL
   // 3. If feed exists → Create subscription only (reuse existing feed)
   // 4. If new feed → Validate feed, create feed, create subscription
   // 5. Enforce 100 feeds/user limit

   // Returns:
   // - 201 Created: Subscription successful
   // - 409 Conflict: User already subscribed to this feed
   // - 400 Bad Request: Invalid feed URL or feed limit reached
   ```

   **For Page:**
   ```go
   // Create content from URL
   content, err := contentService.CreateFromURL(ctx, url, "manual", nil, nil)

   // Add to user's list
   userContent, err := userContentRepo.Create(ctx, &models.UserContent{
       UserID:     userID,
       ContentID:  content.ID,
       Status:     "unread",
   })

   // Return user content with content details
   ```

**New Dependencies:**
- HTTP client for calling Ingest RSS service
- Environment variable: `INGEST_RSS_SERVICE_URL` (default: `http://localhost:8085`)
- Circuit breaker for Ingest RSS service calls (reuse existing pattern)

**Ingest RSS API Integration:**

Endpoint: `POST /api/v1/source/rss/user/{user_id}/subscription`

Request:
```json
{
  "feed_url": "https://example.com/feed.xml"
}
```

Response (Success - 201 Created):
```json
{
  "subscription": {
    "id": "uuid",
    "user_id": "uuid",
    "feed_id": "uuid",
    "subscribed_at": "2025-01-02T10:00:00Z"
  },
  "feed": {
    "id": "uuid",
    "feed_url": "https://example.com/feed.xml",
    "title": "Example Blog",
    "description": "A blog about...",
    "site_url": "https://example.com",
    "polling_tier": "active",
    "status": "active",
    "created_at": "2025-01-02T10:00:00Z",
    "updated_at": "2025-01-02T10:00:00Z"
  },
  "is_new_feed": true
}
```

Error Responses:
- **409 Conflict** (already_subscribed): `{"error": "already_subscribed", "message": "User is already subscribed to this feed"}`
- **400 Bad Request** (feed_limit_reached): `{"error": "feed_limit_reached", "message": "User has reached maximum feed limit of 100"}`
- **400 Bad Request** (invalid_feed): `{"error": "invalid_feed", "message": "Feed URL is not a valid RSS/Atom feed"}`

---

### Phase 3: Detection Endpoint Handler

**File:** `services/read/content/internal/api/handlers/detection_handler.go`

```go
package handlers

type DetectionHandler struct {
    urlDetector service.URLDetector
}

func (h *DetectionHandler) DetectURL(w http.ResponseWriter, r *http.Request) {
    var req dto.DetectURLRequest
    if err := middleware.DecodeJSONBody(r, &req); err != nil {
        middleware.WriteError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
        return
    }

    // Validate URL
    if req.URL == "" {
        middleware.WriteError(w, http.StatusBadRequest, "validation_error", "URL is required", nil)
        return
    }

    // Create context with 10s timeout
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    // Detect URL type
    result, err := h.urlDetector.DetectURL(ctx, req.URL)
    if err != nil {
        // On error, return unknown
        result = &service.URLDetectionResult{
            URL:   req.URL,
            Type:  service.URLTypeUnknown,
            Title: nil,
        }
    }

    middleware.WriteJSON(w, http.StatusOK, result)
}
```

---

### Phase 4: Router Updates

**File:** `services/read/content/internal/api/router.go`

```go
// Add detection handler
detectionHandler := handlers.NewDetectionHandler(urlDetectorService)

r.Route("/api/v1/content", func(r chi.Router) {
    // Existing routes...

    // URL detection endpoint
    r.Post("/detect", detectionHandler.DetectURL)

    // User-content routes
    r.Route("/user/{user_id}", func(r chi.Router) {
        // Updated to handle URL submission with type detection
        r.Post("/", userContentHandler.AddContentToUserFromURL)

        // Existing routes...
    })
})
```

---

## Mobile App Integration

### Phase 5: Mobile Service Updates

**File:** `apps/mobile/src/services/read.ts`

**New Types:**
```typescript
export type URLType = 'feed' | 'page' | 'unknown';

export interface DetectURLResponse {
  url: string;
  type: URLType;
  title: string | null;
}

export interface AddURLRequest {
  url: string;
  type?: URLType;
  title?: string;
}

export interface AddFeedResponse {
  type: 'feed';
  feed_id: string;
  subscription: {
    id: string;
    user_id: string;
    feed_id: string;
    feed_url: string;
    title: string;
    subscribed_at: string;
  };
}

export interface AddPageResponse {
  type: 'page';
  content: UserContentResponse;
}

export type AddURLResponse = AddFeedResponse | AddPageResponse;
```

**New Methods:**
```typescript
/**
 * Detect URL type (feed or page)
 * Non-blocking with 10s timeout
 */
static async detectURL(url: string): Promise<DetectURLResponse> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 10000); // 10s timeout

  try {
    const response = await fetch(
      `${READ_SERVICE_BASE_URL}/api/v1/content/detect`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
        signal: controller.signal,
      }
    );

    clearTimeout(timeoutId);

    if (!response.ok) {
      throw new Error('Detection failed');
    }

    return await response.json();
  } catch (error) {
    clearTimeout(timeoutId);

    // On timeout or error, return unknown
    return {
      url,
      type: 'unknown',
      title: null,
    };
  }
}

/**
 * Add URL to user's content (handles both feeds and pages)
 */
static async addURL(request: AddURLRequest): Promise<AddURLResponse> {
  const userId = await AuthService.getUserId();
  if (!userId) {
    throw new Error('Not authenticated');
  }

  const response = await this.fetchWithAuth(
    `${READ_SERVICE_BASE_URL}/api/v1/content/user/${userId}`,
    {
      method: 'POST',
      body: JSON.stringify(request),
    }
  );

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to add URL: ${error}`);
  }

  return await response.json();
}
```

---

### Phase 6: Mobile UI Component

**File:** `apps/mobile/src/components/AddLinkModal.tsx`

**Component State:**
```typescript
const [url, setUrl] = useState('');
const [detectionResult, setDetectionResult] = useState<DetectURLResponse | null>(null);
const [isDetecting, setIsDetecting] = useState(false);
const [isSubmitting, setIsSubmitting] = useState(false);
```

**Detection Logic:**
```typescript
const handleURLChange = async (newUrl: string) => {
  setUrl(newUrl);

  // Reset previous detection
  setDetectionResult(null);

  // Start non-blocking detection
  if (newUrl.trim()) {
    setIsDetecting(true);

    try {
      const result = await ReadService.detectURL(newUrl);
      setDetectionResult(result);
    } catch (error) {
      console.error('URL detection error:', error);
      // Silently fail - user can still submit
    } finally {
      setIsDetecting(false);
    }
  }
};
```

**Submit Logic:**
```typescript
const handleSubmit = async () => {
  setIsSubmitting(true);

  try {
    const response = await ReadService.addURL({
      url,
      type: detectionResult?.type,
      title: detectionResult?.title,
    });

    if (response.type === 'feed') {
      // Show success message: "Subscribed to feed"
      showSuccessMessage(`Subscribed to ${response.subscription.feed.title}`);
    } else {
      // Show success message: "Article added"
      showSuccessMessage('Article added to reading list');
    }

    onClose();
  } catch (error) {
    // Handle specific error messages
    const errorMessage = error.message || 'Unknown error';

    if (errorMessage.includes('already subscribed')) {
      showErrorMessage('Already subscribed to this feed');
    } else if (errorMessage.includes('Failed to subscribe')) {
      showErrorMessage('Failed to subscribe to feed');
    } else if (errorMessage.includes('Failed to add')) {
      showErrorMessage('Failed to add article');
    } else {
      showErrorMessage(errorMessage);
    }
  } finally {
    setIsSubmitting(false);
  }
};
```

**UI Updates:**
```typescript
// Button text changes based on detection
const submitButtonText = () => {
  if (isSubmitting) return 'Adding...';
  if (detectionResult?.type === 'feed') return 'Add Feed';
  return 'Add';
};
```

---

## Testing Strategy

### Backend Tests

1. **URL Detector Tests:**
   - Test feed detection (RSS, Atom)
   - Test page detection (HTML)
   - Test timeout handling
   - Test invalid URLs
   - Test redirects

2. **Handler Tests:**
   - Test feed submission flow
   - Test page submission flow
   - Test unknown type detection retry
   - Test Ingest RSS service integration
   - Test error handling

### Mobile Tests

1. **Service Tests:**
   - Test detection timeout (10s)
   - Test detection error handling
   - Test submission with type hint
   - Test submission without type hint

2. **UI Tests:**
   - Test non-blocking detection
   - Test button text updates
   - Test submission during detection
   - Test submission after detection

---

## Migration Notes

### Breaking Changes
- The existing `POST /api/v1/content/user/{user_id}` endpoint will be **replaced**
- Old request format: `{content_id: uuid}` (requires pre-created content)
- New request format: `{url: string, type?: string, title?: string}` (one-step flow)

### Backward Compatibility Option
If needed, we can support both formats by checking the request body:
```go
// If request has content_id field, use old flow
// If request has url field, use new flow
```

### Database Changes
- No database migrations required
- Leverages existing tables in Content and Ingest RSS services

---

## Implementation Order

1. ✅ **Phase 1:** URL Detector Service (core detection logic)
2. ✅ **Phase 2:** Detection Endpoint Handler
3. ✅ **Phase 3:** Enhanced User Content Handler (URL submission)
4. ✅ **Phase 4:** Router Updates
5. ✅ **Phase 5:** Mobile Service Updates
6. ✅ **Phase 6:** Mobile UI Component
7. ✅ **Phase 7:** Integration Testing
8. ✅ **Phase 8:** Documentation Updates
   - OpenAPI specification (services/read/content/api/openapi.yaml)
   - CLAUDE.md API endpoints section
   - URL_DETECTION_PLAN.md completion status

---

## Answered Questions

1. **Feed Subscription Management:**
   - ✅ The Ingest RSS service manages user-feed relationships through the `feed_subscriptions` table
   - ✅ Feed subscription limit (100 feeds/user) is enforced by Ingest RSS service
   - ✅ Content service delegates to Ingest RSS service API

2. **Duplicate Feed Handling:**
   - ✅ If user already subscribed: Return "already_subscribed" error (409 Conflict)
   - ✅ If another user subscribed to same feed: Ingest RSS service handles this gracefully:
     - Feed already exists in database
     - Creates new subscription for current user
     - Does NOT create duplicate feed

3. **Feed Preview:**
   - ✅ No preview needed in detection response

4. **Error Messages:**
   - ✅ "Already subscribed to this feed" (when user re-subscribes)
   - ✅ "Failed to subscribe to feed" (generic error for other failures)
   - ✅ "Failed to add article" (for page fetch failures)

5. **Analytics:**
   - ✅ No analytics tracking required at this time
