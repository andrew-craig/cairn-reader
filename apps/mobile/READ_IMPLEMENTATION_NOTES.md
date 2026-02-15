# Read Page Implementation Notes

## Overview
The Read page has been implemented to connect to the Read service backend. The implementation follows the same patterns as the Explore screen and includes full pagination support.

## What Was Implemented

### 1. Backend API Client
- Created `src/services/read.ts` with the `ReadService` class
- Implements all Read service API endpoints:
  - `listUserContents()` - List user's saved articles with pagination
  - `searchUserContents()` - Full-text search across saved content
  - `addContentToUser()` - Add article to reading list
  - `updateUserContent()` - Update metadata (status, favorite, scroll position)
  - `deleteUserContent()` - Remove article from reading list
  - `transformToArticle()` - Transform backend response to mobile Article format

### 2. TypeScript Types
- Created `src/types/read.ts` with all necessary types:
  - `ContentResponse` - Backend content structure
  - `UserContentResponse` - User-specific content metadata
  - `UserContentsListResponse` - Paginated list response
  - `ContentStatus` - Type for read status ('unread' | 'reading' | 'completed' | 'archived')
  - Request/response types for all API operations

### 3. ReadScreen Implementation
- Updated `src/screens/ReadScreen.tsx` to fetch from backend API
- Implemented pagination (20 items per page)
- Pull-to-refresh functionality
- Infinite scroll with loading indicator
- Error handling with user alerts
- Proper loading states

### 4. Configuration
- Updated `src/config/api.ts` to include `READ_SERVICE_URL`
- Exported new types from `src/types/index.ts`
- Exported ReadService from `src/services/index.ts`

## Backend API Endpoints Used

All endpoints are already implemented in the Read service:

```
GET  /api/v1/users/{user_id}/contents          - List user's content (with pagination)
POST /api/v1/users/{user_id}/contents          - Add content to user
GET  /api/v1/users/{user_id}/contents/search   - Search user's content
PATCH /api/v1/users/{user_id}/contents/{id}    - Update content metadata
DELETE /api/v1/users/{user_id}/contents/{id}   - Delete content from user
```

## What Still Needs Implementation

### 1. Article Detail Screen
The current implementation has a placeholder for article navigation:
```typescript
const handleArticlePress = (article: Article) => {
  // TODO: Navigate to article detail screen
  console.log('Article pressed:', article.id);
};
```

**Required:**
- Create `ArticleDetailScreen.tsx` to display article content
- Fetch full article content including cleaned HTML
- Display reading position and allow scrolling
- Add navigation from ReadScreen to ArticleDetailScreen

### 2. Add Article Functionality
The add button currently has a placeholder:
```typescript
const handleAddPress = () => {
  // TODO: Navigate to add article screen or show add modal
  console.log('Add pressed');
};
```

**Required:**
- Create modal or screen to add articles manually
- Allow user to paste URL
- Call `ReadService.addContentToUser()` with the URL
- Refresh article list after adding

### 3. Search Functionality
The search button currently has a placeholder:
```typescript
const handleSearchPress = () => {
  // TODO: Navigate to search screen
  console.log('Search pressed');
};
```

**Required:**
- Create search screen or modal
- Implement search input with debouncing
- Call `ReadService.searchUserContents()` with query
- Display search results

### 4. Article Actions
Users should be able to:
- Mark articles as favorite (swipe action or button)
- Delete articles (swipe action)
- Change article status (unread → reading → completed → archived)

**Implementation:**
- Add swipe actions to ArticleRow component
- Call `ReadService.updateUserContent()` for status/favorite changes
- Call `ReadService.deleteUserContent()` for deletion
- Update local state optimistically

### 5. Filter/Sort Options
The design shows filtering/sorting controls that are not yet implemented:
- Filter by status (unread, reading, completed, archived)
- Filter by favorites
- Sort by date added, title, etc.

**Implementation:**
- Add filter UI in header
- Update `ReadService.listUserContents()` calls with filter params
- Persist filter preferences

### 6. Sync with Explore Service
Currently, articles from the Explore service are separate from the Read service. Need to implement:
- When user saves article from Explore, add to Read service
- Integration point in ExploreScreen to call `ReadService.addContentToUser()`

## Backend Service Notes

### Content Service (Port 8083)
The Read service content API is already fully implemented with:
- ✅ User content listing with pagination
- ✅ Full-text search
- ✅ Add/update/delete operations
- ✅ Content extraction and storage
- ✅ Duplicate detection

### What Works Out of the Box
All backend functionality is ready:
1. Articles are stored with cleaned HTML content
2. User-specific metadata (status, favorites, scroll position)
3. Full-text search across title, author, description
4. Pagination with 20 items per page
5. Content deduplication by hash

## Testing Checklist

Before testing with the mobile app, ensure:

1. **Backend Services Running:**
   ```bash
   cd services/read
   docker compose up
   ```

2. **User Authentication:**
   - User must be logged in (access token required)
   - ReadService uses same auth flow as ExploreService

3. **Test Scenarios:**
   - [ ] Initial load shows empty state
   - [ ] Add test content via API
   - [ ] Pull to refresh works
   - [ ] Pagination loads more items
   - [ ] Error handling works (network errors, auth errors)

## API Configuration

For local testing, update `apps/mobile/src/config/api.ts`:

```typescript
export const API_CONFIG = {
  USER_SERVICE_URL: 'http://localhost:8080',
  RECOMMENDER_SERVICE_URL: 'http://localhost:8081',
  READ_SERVICE_URL: 'http://localhost:8083',  // Content Service port
  REQUEST_TIMEOUT: 30000,
};
```

For production, the current configuration uses:
```typescript
READ_SERVICE_URL: 'https://cairn.seatrain.net'
```

## Next Steps

Priority order for remaining work:

1. **Article Detail Screen** - Most important for MVP functionality
2. **Add Article Modal** - Allows users to save articles manually
3. **Article Actions** - Swipe to delete/favorite
4. **Search Screen** - Full-text search UI
5. **Filter/Sort** - Enhance browsing experience
6. **Explore Integration** - Connect Explore and Read services

## Files Modified

- ✅ `src/types/read.ts` - New file with backend types
- ✅ `src/services/read.ts` - New file with ReadService client
- ✅ `src/screens/ReadScreen.tsx` - Updated to use backend API
- ✅ `src/config/api.ts` - Added READ_SERVICE_URL
- ✅ `src/types/index.ts` - Export read types
- ✅ `src/services/index.ts` - Export ReadService

## Related Documentation

- [Read Service OpenAPI Spec](/services/read/api/openapi.yaml)
- [Read Service README](/services/read/README.md)
- [Read Service Implementation Plan](/services/read/IMPLEMENTATION_PLAN.md)
- [Engineering Principles](/docs/ENGINEERING_PRINCIPLES.md)
