# Cairn Explore Requirements

## Overall Goals

The objective of the Cairn Explore services is to collect long-form web content and recommend it to the user. For the initial implementation, this will include:
* Collecting articles from the feeds in the [Kagi Small Web Text collection](https://github.com/kagisearch/smallweb/blob/main/smallweb.txt)
* Storing those articles in a database
* Serving a recommendation of the next five articles for a user to read

## Structure

The project will be composed of two services:
* **Fetcher** - manages the collection of articles from the feeds
* **Recommendation Engine** - stores the content of the articles, and makes recommendations to the user

**Architecture Principle**: Services communicate via HTTP APIs only. The fetcher NEVER accesses the database directly.

## Fetcher Requirements

### 1. Maintain Source List
- Store the list of sources in a database (managed by the Recommendation Engine)
- Fetch the latest version of the [Kagi Small Web Text collection](https://github.com/kagisearch/smallweb/blob/main/smallweb.txt) once per day
- When syncing feeds from Kagi:
  - **Only add new feeds** (never modify or delete existing feeds)
  - Match feeds by URL to detect duplicates
  - New feeds should be enabled by default
- Disabled feeds remain disabled until manual re-enable (daily sync does not auto-enable failed feeds)

### 2. Fetch the Latest Content
- Once per minute, identify the feed with the longest time since an update and fetch the content of that feed
- Prioritization rules:
  1. **Never-fetched feeds first** (feeds where `last_fetched_at` is NULL)
  2. Then oldest `last_fetched_at` timestamp
- Only fetch enabled feeds (`enabled = true`)
- Article filtering:
  - **Only send new articles** to the recommendation engine
  - Use `updated_at` or `published_at` timestamps from the feed to determine newness
  - Comparison doesn't need to be perfect (reasonable best-effort)
- Maintain a record of fetch failures:
  - Track `consecutive_failures` counter per feed
  - **Disable a feed** after 10 consecutive failures
  - Reset counter to 0 on successful fetch

## Recommendation Engine Requirements

### 1. Maintain a Database of Articles
- Provide an endpoint for the fetcher to pass in articles
- Store the content of the articles in a database
- Required article fields:
  - Basic metadata (title, link, content, published_at, etc.)
  - Engagement metrics (upvotes, downvotes, recommends counters)
  - `deleted` boolean flag (default: false)
  - Foreign key to feeds table

### 2. Article Database Clean-up
- After 90 days, **hard delete** articles from the database (based on `published_at` timestamp)
- This is a permanent removal, not a soft delete

### 3. Recommendation Algorithm
- Provide an API that can be called with a `user_id` that returns 5 recommended articles
- Each time an article is recommended, increment the `recommends` counter

#### Article Selection Logic:
1. **4 high-quality articles** (articles with `recommends > 0`):
   - Calculate quality score: `(upvotes - (downvotes * 3)) / recommends`
   - Select 4 articles with highest quality scores
   - Exclude articles where `deleted = true`
   - Do NOT exclude articles user has voted on

2. **1 exploration article** (articles with `recommends < 100`):
   - Randomly select from all articles with `recommends < 100`
   - No prioritization based on recommend count within this pool
   - Exclude articles where `deleted = true`

3. **Handling edge cases**:
   - If fewer than 5 total eligible articles exist, fill remaining slots with high-quality articles
   - No duplicate prevention: the same article may be recommended to the same user multiple times

### 4. Upvote / Downvote System
- Users can upvote or downvote articles
- The Recommendation Engine provides an API that enables the app to report upvotes and downvotes
- Vote management rules:
  - Track votes per user (one vote per user per article)
  - Users can **toggle their vote directly** (upvote→downvote or vice versa in one API call)
  - When toggling: adjust BOTH counters (e.g., upvote→downvote: `upvotes -= 1`, `downvotes += 1`)
  - Users can remove their vote entirely
  - Prevent double-voting (UNIQUE constraint on user_id + article_id in votes table)
  

