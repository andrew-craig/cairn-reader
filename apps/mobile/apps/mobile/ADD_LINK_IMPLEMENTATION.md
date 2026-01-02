# Add Link Modal Implementation

## Overview
The Add Link functionality allows users to add articles or feeds to their Read list by entering a URL. The modal provides two options:
1. **Add** - Directly save an article from the URL
2. **Find feed** - Scan the URL for RSS/Atom feeds (placeholder for future implementation)

## Components

### AddLinkModal
Location: [src/components/AddLinkModal.tsx](src/components/AddLinkModal.tsx)

A bottom sheet modal that appears when the user taps the "+" button in the Read screen.

**Features:**
- URL validation with automatic `https://` prefix
- Error handling and user feedback
- Loading states during API calls
- Keyboard-aware layout (works on both iOS and Android)
- Dark mode support
- Accessible touch targets (48px minimum height)

**Props:**
```typescript
interface AddLinkModalProps {
  visible: boolean;
  onClose: () => void;
  onAddArticle: (url: string) => Promise<void>;
  onFindFeed: (url: string) => Promise<void>;
}
```

### ReadScreen Integration
Location: [src/screens/ReadScreen.tsx](src/screens/ReadScreen.tsx)

The ReadScreen has been updated to:
- Show the AddLinkModal when the "+" button is pressed
- Handle article addition via the Read Service API
- Refresh the article list after successful addition
- Display success/error messages to the user

## User Flow

### Adding an Article
1. User taps the "+" button in the Read screen header
2. Modal appears with a text input for the URL
3. User enters a URL (e.g., `example.com/article` or `https://example.com/article`)
4. User taps "Add"
5. App validates the URL and adds `https://` if needed
6. App sends the URL to the Read Service API
7. On success: Modal closes, success message shown, article list refreshes
8. On error: Error message displayed in the modal

### Finding a Feed (Placeholder)
1. User enters a URL for a website
2. User taps "Find feed"
3. Currently shows a "Coming Soon" alert
4. Future: Will scan the page for RSS/Atom feeds and present options

## URL Validation

The modal includes smart URL handling:
- Accepts URLs with or without `https://` prefix
- Automatically adds `https://` if no protocol is specified
- Validates URLs using the native URL parser
- Shows error messages for invalid URLs

**Examples of valid inputs:**
- `example.com/article` → `https://example.com/article`
- `https://example.com/article` → `https://example.com/article`
- `http://example.com/article` → `http://example.com/article`

## Styling

The modal follows the Figma design specifications:
- Position: Bottom sheet (from bottom of screen)
- Size: Full width, auto height based on content
- Background: Theme-aware (light/dark mode)
- Border radius: 16px on top corners
- Padding: 16px horizontal, 24px top, 24px/32px bottom (iOS has extra for safe area)

**Button styles:**
- **Add button**: Primary style (dark background, white text)
- **Find feed button**: Secondary style (card background, bordered)
- Both buttons are full width in a horizontal layout
- 48px minimum height for accessibility

## API Integration

The modal uses the ReadService to add content:

```typescript
await ReadService.addContentToUser({
  url: normalizedUrl,
  source_type: 'manual',
});
```

This calls the Read Service API endpoint:
```
POST /api/v1/users/{userID}/contents
Body: { url: string, source_type: 'manual' }
```

## Error Handling

The implementation includes comprehensive error handling:
- **URL validation errors**: Shown inline in the modal
- **API errors**: Caught and displayed in the modal
- **Network errors**: Passed through from ReadService with meaningful messages
- **Authentication errors**: Handled by ReadService with automatic token refresh

## Future Enhancements

### Feed Discovery (TODO)
The "Find feed" functionality is planned for a future update. It will:
1. Fetch the HTML content from the provided URL
2. Parse the HTML to find `<link>` tags with `type="application/rss+xml"` or `type="application/atom+xml"`
3. Present a list of discovered feeds to the user
4. Allow the user to subscribe to one or more feeds

**Implementation considerations:**
- May require a backend endpoint to avoid CORS issues
- Should handle common feed formats (RSS 2.0, Atom, JSON Feed)
- Could use the RSS Fetcher Service's feed validation
- Should support both explicit feed URLs and webpage URLs

### Additional Features
- **Browser extension integration**: Share URLs directly from a browser
- **Smart detection**: Automatically determine if URL is an article or feed
- **Bulk import**: Support adding multiple URLs at once
- **URL preview**: Show article title/image before adding
- **Recent URLs**: Keep history of recently added URLs

## Testing

To test the Add Link functionality:

1. Start the mobile app:
   ```bash
   cd apps/mobile
   npm start
   ```

2. Navigate to the Read tab

3. Tap the "+" button in the header

4. Try adding these test URLs:
   - Valid article: `https://paulgraham.com/weird.html`
   - Without protocol: `paulgraham.com/weird.html`
   - Invalid URL: `not a url` (should show error)

5. Verify:
   - Modal appears and dismisses correctly
   - URL validation works
   - Success message appears after adding
   - Article appears in the list after refresh
   - Error messages display for failures

## Known Limitations

1. **Feed discovery not implemented**: "Find feed" button shows a placeholder alert
2. **No URL preview**: Articles are added without showing a preview
3. **No duplicate detection UI**: Backend handles duplicates, but no UI feedback
4. **No feed subscription UI**: Only articles can be added via this modal
