# Cairn Product Requirements

## Product Overview

Cairn is a read-it-later mobile application that helps users discover and read long-form content. The app combines two core experiences:

1. **Explore**: Discover curated articles from across the web
2. **Read**: Subscribe to RSS feeds and build a personal reading library

Target Users: People who want to read thoughtful, long-form content

## Product Requirements - Reading Experience (for both Explore and Read)

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
- **Reading Status**: Articles can be unread, read, or archived. Statuses will be synced across devices
- **Scroll Position**: Reading position automatically saved
  - Resume reading on any device
  - Character-based position for consistent experience across screen sizes
- **Favorites**: Star articles for quick access. Favorites synced across devices
- **Offline availability**: saved content is stored on device for access offline. Changes are synced when the connection is restored


## Product Requirements - Content Discovery (Explore)

#### Article Curation
- Content will ffocus on independent blogs, personal websites, and thoughtful writing
  - Articles will initially be sourced from [Kagi Small Web Text collection](https://github.com/kagisearch/smallweb/blob/main/smallweb.txt)
- Content will be removed after 90 days

#### Article Recommendations
- Users receive personalized article recommendations
- Recommendations balance quality content with exploration. Initial split is:
  - 80% high-quality articles based on community engagement
  - 20% exploratory article to discover new content
- Fresh content added continuously (new articles fetched every minute)
- Content sourced from curated collection of independent blogs and websites

#### Engagement & Feedback
- **Upvote/Downvote**: Users can vote on articles to improve recommendations
- **Vote Management**:
  - Change vote from upvote to downvote (or vice versa)
  - Remove vote entirely
  - One vote per article per user
- Voting influences future recommendations for all users


## Product Requirements - Personal Reading Library (Read)

The personal reading library (Req 2.1) allows users to maintain a list of must-read content that combines
- content they have saved to read later (Req 2.2)
- content from trusted sources that they know they will want to read from RSS feeds (Req. 2.3), email newsletters (Req 2.4) or other sources

#### 2.1 Personal Library

Users will have a personal library of content which they can manage. This library will be accessible as a list of articles/content in the Read tab of the app.

Users will be able to:
- **Search** the list of articles (initially title and author, but target state includes full text)
- **Filter** by status (unread, read, archived) and favorites. The default will be to show all non-archived content
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
  - Re-enable or remove disabled feeds

Content extraction
- If full article is not available in the RSS feed, we will attempt to extract it from the source. If this fails, we will fallback to RSS summary if full article unavailable

Secondary requirements
- Update handling
  - **Update Detection**: System detects when articles are updated at source
  - **Automatic Updates**: Updated articles refreshed automatically
  - **Preserved Progress**: Reading position maintained when content updates

#### 2.4 Ingestion - Email

#### 2.5 Content Management
- **Delete Articles**: Remove articles from personal reading list
- **Notes**: Add personal notes to articles (future enhancement)
- **Tags**: Organize with custom tags (future enhancement)


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
2. Browse recommended articles (Explore)
3. Vote on articles to personalize recommendations
4. Subscribe to favorite RSS feeds (Read)

### Daily Reading Routine
1. Check Explore for new recommendations
2. Read and vote on interesting articles
3. Browse new articles from RSS subscriptions
4. Mark articles as read/archived
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
- Upvote/downvote ratio
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
- Web interface (mobile-only initially)
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
- Popular articles from community
- Trending content alerts
- Feed discovery engine

### Mobile Features
- Offline reading mode
- Download for offline access
- Dark mode reading
- Text-to-speech
- Adjustable fonts and spacing
- Reader themes

### Technical Improvements
- Web interface
- Browser extensions (save from web)
- Email-to-Cairn (forward articles via email)
- API for third-party integrations
- Import from Pocket, Instapaper, etc.

## Design Principles

1. **Simplicity First**: Focus on core reading experience without feature bloat
2. **Privacy Focused**: User data is private and secure
3. **Quality Over Quantity**: Curated sources and quality scoring
4. **Reliable Sync**: Reading progress always in sync
5. **Respectful of Time**: Clean content, no distractions
6. **Open & Portable**: RSS-based, export capabilities (future)
7. **Community-Driven**: Upvotes/downvotes improve recommendations for everyone

## Platform Requirements

### Mobile App
- iOS (React Native/Expo)
- Android (React Native/Expo)
- Responsive to various screen sizes
- Works on tablets and phones

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

1. Should users be able to see which articles they've already voted on?
2. Should there be a limit on how many articles can be saved?
3. Should feeds be shareable between users?
4. Should there be categories or topics for Explore content?
5. Should users be able to report problematic content?
6. Should there be a "read later" feature separate from subscriptions?
