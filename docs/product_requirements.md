# Cairn Product Requirements

## Product Overview

Cairn is a read-it-later application (mobile and web) that helps users read long-form content. Content lands in one of two lists:

1. **Feed**: a list to skim. Items are temporary
2. **Reads**: a personal reading library of content to read or triage

The app's tabs are **Feed | Reads | You**. Removed: Explore (see `docs/archive/explore_product_requirements.md`).

Target Users: People who want to read thoughtful, long-form content

## Product Requirements - Reading Experience (for both Feed and Reads)

The core principles of the reading experience are:
- **Optimise for readability**: Articles processed for optimal readability
- **Reading progress saved and synced**
- **Original Source**: Link to original article always available

Optimise for readability
- Removes ads, navigation, and clutter
- Preserves images, formatting, and structure
- Article images displayed inline
- All HTML sanitized to prevent security issues

Reading progress saved and synced
- **Reading Status**: Articles can be unread, reading, completed, or archived. Statuses will be synced across devices
- **Scroll Position**: Reading position automatically saved
  - Resume reading on any device
  - Character-based position for consistent experience across screen sizes
- **Favorites**: Star articles for quick access. Favorites synced across devices
- **Offline availability** (mobile only — shipped): saved content is stored on device (SQLite) for access offline. Reading-status, favorite, scroll-position, and archive changes made offline are queued and synced when the connection is restored
  - Non-goals: web/PWA offline access (out of scope for the web app, see `docs/detailed_requirements/web_app_requirements.md`); image caching for offline viewing (tracked separately as image hosting/optimization); backend delta-sync or ETag support; server-side conflict resolution — v1 sync is last-write-wins; manual per-article "download for offline" control (article bodies are prefetched automatically, not user-triggered)


## Product Requirements - Personal Reading Library (Feed and Reads)

The personal library (Req 2.1) holds content from the sources a user follows and the links they save. It is split into two lists (Req 2.6). It combines
- content they have saved to read later (Req 2.2)
- content from trusted sources that they know they will want to read from RSS feeds (Req. 2.3), email newsletters (Req 2.4) or other sources

#### 2.1 Personal Library

Users will have a personal library of content which they can manage. The library is accessible as two lists of articles/content, in the Feed and Reads tabs of the app (see Req 2.6).

Users will be able to:
- **Search** the list of articles (initially title and author, but target state includes full text)
- **Filter** by status (unread, reading, completed, archived) and favorites. The default will be to show all non-archived content
- **Add** new content by either saving an article, or adding a new source

Opening an article will the app's reading experience (see above)


#### 2.2 Saving - Read Later

- Graceful handling of paywalled or restricted content


#### 2.3 Ingestion - RSS Feeds
- **Subscribe to Feeds**: Add any RSS feed via URL
- **Auto-Delivery**: New articles automatically appear in reading list
  - service silently and automatically retries on fetch failures
- Manage feeds:
  - View all subscribed feeds
  - Unsubscribe from feeds
  - Feeds auto-disable after 7 consecutive days of errors
  - Re-enable or remove disabled feeds (the apps do not yet offer re-enable; tracked as a gap)
- **Choose the list**: each subscription delivers to Feed or Reads (Req 2.6). The default is Reads

Content extraction
- If full article is not available in the RSS feed, we will attempt to extract it from the source. If this fails, we will fallback to RSS summary if full article unavailable

Secondary requirements
- Update handling
  - **Update Detection**: System detects when articles are updated at source
  - **Automatic Updates**: Updated articles refreshed automatically
  - **Preserved Progress**: Reading position maintained when content updates

#### 2.4 Ingestion - Email
- Each user has one ingest email address (get or create on demand). Newsletters forwarded to it become articles
- Each sender is tracked as a source (email address, name, count of emails, last received)
- Content is cleaned (tracking pixels, unsubscribe footers and preheader text removed) and sanitized before delivery
- **Choose the list**: each sender delivers to Feed or Reads (Req 2.6). New senders default to Reads
- Delivery to the Content Service is retried automatically on failure

See `services/read/email/CLAUDE.md` for the pipeline.

#### 2.5 Content Management
- **Delete Articles**: Remove articles from personal reading list
- **Notes**: Add personal notes to articles (future enhancement)
- **Tags**: Organize with custom tags (future enhancement)

#### 2.6 Feed and Reads Lists
- Every saved item is in exactly one list: **Feed** (skim) or **Reads** (read or triage)
- The user picks the list per source (RSS subscription or email sender), on the RSS and Newsletters pages and when adding a feed
- A source with no choice delivers to Reads: this is the default for new RSS subscriptions and new email senders
- Links the user saves directly always go to Reads
- Changing a source's list affects new items only; existing items stay where they are
- Non-favorited Feed items are deleted 30 days after they were added. Favorited Feed items are kept, and Reads items are never expired. Removing the favorite from an item older than 30 days makes it eligible for deletion at the next daily cleanup


## Product Requirements - Profile > Content

## Product Requirements - Settings



### 5. Reliability & Performance

#### Content Deduplication
- Same article from same feed stored only once
- Multiple users can save the same content
- Shared storage with individual user metadata

## User Workflows

### New User Onboarding
1. Sign up / authenticate
2. Subscribe to favorite RSS feeds and choose Feed or Reads for each

### Daily Reading Routine
1. Skim the Feed for new items, favoriting any to keep
2. Read the Reads list
3. Browse new articles from RSS subscriptions
4. Mark articles as completed/archived
5. Search for specific topics in reading library

### Feed Discovery & Management
1. Discover new blogs/websites
2. Add RSS feed URL
3. New articles automatically appear
4. Manage subscriptions (view, disable, unsubscribe)
5. Re-enable feeds if needed

## Success Metrics

### Engagement
- Articles read per day per user
- Time spent reading
- Return rate (daily active users)
- Feed subscriptions per user

### Content Quality
- Article completion rate
- Favorite rate
- Search usage

### System Health
- Feed fetch success rate
- Content extraction success rate
- Feed auto-disable frequency
- Average response time

## Out of Scope (Initial Release)

The following features are **not** included in the initial release:

- OBSERVABILITY METRICS
- ALERTING INFRASTRUCTURE
- Social features (sharing, comments, collaborative reading)
- Image downloading/hosting (images loaded from source)
- Advanced search (full content search, filters)
- Content recommendations based on individual reading history
- Import from other services (Pocket, Instapaper, etc.)
- Export functionality (Markdown, PDF, EPUB)
- Reading analytics and statistics
- Email notifications
- Custom collections/folders
- Highlights and annotations
- Tags and categorization

## Future Enhancements

### Content Features
- Full-text content search (search within article bodies)
- Related article suggestions
- Reading time estimates
- Content archiving (snapshot articles at save time)
- Multiple content sources (web scraper, email forwarding, Pocket import)
- Export to various formats (Markdown, PDF, EPUB)

### User Features
- Highlights and annotations
- Personal notes and comments
- Custom tags and collections
- Reading statistics and streaks
- Content sharing between users
- Collaborative reading lists

### Discovery Features
- Personalized recommendations based on reading history
- Topic-based browsing
- Feed discovery engine

### Mobile Features
- Dark mode reading
- Text-to-speech
- Adjustable fonts and spacing
- Reader themes

### Technical Improvements
- Browser extensions (save from web)
- API for third-party integrations
- Import from Pocket, Instapaper, etc.

## Design Principles

1. **Simplicity First**: Focus on core reading experience without feature bloat
2. **Privacy Focused**: User data is private and secure
3. **Quality Over Quantity**: Curated sources and quality scoring
4. **Reliable Sync**: Reading progress always in sync
5. **Respectful of Time**: Clean content, no distractions
6. **Open & Portable**: RSS-based, export capabilities (future)

## Platform Requirements

### Mobile App
- iOS (React Native/Expo)
- Android (React Native/Expo)
- Responsive to various screen sizes
- Works on tablets and phones

### Web App
- Browser client (React/Vite) with the same Feed, Reads and You structure. See `docs/detailed_requirements/web_app_requirements.md`

### Authentication
- Email/password signup
- Mobile device authentication (Expo device ID)
- Secure token-based authentication (JWT)
- Account upgrade from device-only to email/password

### Performance
- Fast app startup
- Smooth scrolling
- Quick article loading
- Responsive UI (no blocking operations)

## Privacy & Security

### User Privacy
- No tracking of reading behavior for advertising
- No selling of user data
- No third-party analytics (initially)
- User controls their own data

### Content Security
- HTML sanitization prevents XSS attacks
- Safe link validation
- Content Security Policy in WebViews
- No execution of external scripts

### Account Security
- Secure password storage (bcrypt)
- JWT tokens with refresh rotation
- HTTPS only in production
- Secrets managed via HashiCorp Vault

## Accessibility

- Readable content formatting
- Adjustable text sizes (future)
- Dark mode support (future)
- Screen reader compatibility (future)
- Keyboard navigation (web, future)

## Localization

- Initial release: English only
- Future: Support for multiple languages
- Future: Right-to-left language support

## Legal & Compliance

- Fair use of RSS content (attribution to original sources)
- Compliance with robots.txt
- Respect for source site terms of service
- GDPR compliance for EU users (future)
- Data export capabilities (future)

## Open Questions

1. Should there be a limit on how many articles can be saved?
2. Should feeds be shareable between users?
3. Should users be able to report problematic content?

Removed: Explore-related questions (see `docs/archive/explore_product_requirements.md`).
