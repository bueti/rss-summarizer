# RSS Summarizer

Service that fetches RSS feeds, downloads the text, sends it to an LLM to summarize it, and rates it by importance. Designed for personal use to cut through information overload and surface what matters.

## Overview

The RSS Summarizer helps users stay informed without drowning in content. Instead of manually checking dozens of feeds, users get AI-powered summaries with intelligent filtering based on importance and topics of interest.

### Target Users
- **Information workers** who follow many blogs, news sites, and industry publications
- **Researchers** tracking specific topics across multiple sources
- **Tech enthusiasts** following personal blogs, Hacker News, Reddit, etc.

## User Stories

**As a user, I want to:**
- Add RSS feeds by URL so I can aggregate content from my favorite sources
- See summarized articles so I can quickly scan what's new without reading full posts
- Filter by importance rating so I can focus on the most relevant content first
- View articles by topic so I can find content related to my interests
- Mark summaries as read/unread to track what I've reviewed
- **Save articles for later reading** so I can quickly access my must-reads
- **Archive articles** to keep them stored but remove from main view
- Click through to original articles when a summary catches my interest
- Customize my topic preferences so the system learns what's important to me

## Tech Stack

### Backend (Go + Huma)
- **API Framework**: Huma v2 for auto-generated OpenAPI docs and type-safe endpoints
- **Orchestration**: Temporal for FSM-based workflow orchestration
  - Feed polling workflow (scheduled, fault-tolerant)
  - Article processing workflow (fetch → extract → summarize → rate)
- **Database**: PostgreSQL for relational data (feeds, articles, users)
- **LLM Integration**: OpenAI API or Anthropic API for summarization and rating
- **RSS Parser**: Go library (e.g., `gofeed`) for parsing RSS/Atom feeds
- **Web Scraper**: For extracting article content when RSS only has excerpts

### Frontend (Svelte 5 + Tailwind)
- **Framework**: Svelte 5 with runes for reactive state
- **Styling**: Tailwind CSS for utility-first styling
- **Routing**: SvelteKit for SSR and routing
- **State Management**: Svelte stores for global state
- **HTTP Client**: Fetch API with typed endpoints

### Infrastructure
- **Containerization**: Docker + Docker Compose for local development
- **Authentication**: OAuth 2.0 (Google provider via library like `coreos/go-oidc`)
- **Deployment**: Docker containers (can deploy to VPS, cloud, or self-host)

## Features

### 1. Authentication
- **OAuth Login**: Google OAuth for secure, passwordless authentication
- **Session Management**: JWT tokens for API authentication
- **Personal Use**: Single-user or small group (no complex multi-tenancy)

### 2. Feed Management
- **Add Feed**: User provides RSS/Atom feed URL
  - System validates URL and fetches feed metadata (title, description)
  - Optional: Auto-detect feed URL from website URL
- **Edit Feed**: Update feed name, polling frequency, or archive old items
- **Delete Feed**: Remove feed and optionally keep/delete historical articles
- **Feed List View**: Display all feeds with article counts and last update time

### 3. Content Processing Pipeline

**Workflow (Temporal FSM):**
```
Poll Feed → Fetch New Items → Extract Full Text → Summarize → Rate Importance → Tag Topics
```

- **Polling**: Background job that checks feeds at configurable intervals
  - Uses feed-specific `PollFrequency` if set, otherwise falls back to user's `DefaultPollInterval`
  - User can set global default (e.g., 30 minutes) in preferences
  - Individual feeds can override (e.g., high-priority feed polls every 15 minutes)
- **Text Extraction**: For feeds with only excerpts, scrape full article content
- **Summarization**: LLM generates 2-3 sentence summary + key points
- **Importance Rating**: LLM rates 1-5 based on user's topic preferences
- **Topic Tagging**: Auto-detect 1-2 broad, top-level topics per article using a 3-layer defense system

  **Philosophy**: Topics should be broad, top-level categories (e.g., "Go", "Security", "AI") that help organize content without creating noise. Avoid specific variations like "Go 1.21", "Application Security", or "Machine Learning Techniques".

  **Layer 1 - LLM Prompt Constraints**:
  - LLM instructed to generate only 1-2 topics maximum (prefer 1)
  - Provided with whitelist of approved categories:
    * Languages: Go, Rust, Python, JavaScript, TypeScript, Java, C++
    * Cloud/Infra: Kubernetes, Docker, Cloud, AWS, GCP, Azure
    * Domains: Security, AI, Databases, Web, APIs, DevOps
    * General: Engineering, Architecture, Performance, Testing, Git, Linux, Open Source
  - Explicit rules: Use single words when possible, no version numbers, no multi-word descriptions
  - Examples provided in prompt showing good vs bad topics

  **Layer 2 - Server-Side Normalization**:
  - Even if LLM generates specific topics, they are automatically mapped to broad categories:
    * "Machine Learning", "Deep Learning", "LLM", "ChatGPT" → "AI"
    * "PostgreSQL", "MySQL", "MongoDB", "SQL" → "Databases"
    * "React", "Frontend Development", "Backend Development" → "Web"
    * "REST API", "GraphQL", "API Development" → "APIs"
    * "CI/CD", "Continuous Integration", "DevOps" → "DevOps"
    * "Cybersecurity", "Application Security", "Network Security" → "Security"
    * 60+ additional mappings covering common variations
  - Topics normalized to Title Case for consistency
  - Maximum 2 topics enforced after deduplication

  **Layer 3 - Database Constraints**:
  - Case-insensitive unique constraint: `CREATE UNIQUE INDEX ON topics (user_id, LOWER(name))`
  - Prevents duplicates like "AI", "ai", "Ai" from being stored
  - All topic lookups are case-insensitive: `WHERE LOWER(name) = LOWER($1)`
  - Unused auto-detected topics automatically cleaned up via periodic job

  **Benefits**:
  - Prevents topic clutter (15-20 topics total vs hundreds)
  - Makes topic filtering actually useful
  - Sustainable - new articles don't create new topics constantly
  - Better content discovery through broad categorization

  **Implementation Details**:
  - Topic mapping logic in `backend/internal/service/llm/llm.go` (`normalizeTopics()` function)
  - Database migrations 007-009 handle consolidation of existing topics
  - See `backend/TOPIC_CLEANUP.md` for full documentation on topic management
  - Approved topic categories can be extended by updating all 3 layers

- **Error Handling**: Failed feeds increment `ErrorCount`; after 3 consecutive failures, status changes to "error"

### 4. Smart Filtering & Discovery

- **Importance Ratings**:
  - 5 = Must read
  - 4 = High priority
  - 3 = Medium interest
  - 2 = Low priority
  - 1 = Likely skip
- **Topic Detection**: Auto-generate topics from article content
- **Topic Preferences**: User can mark topics as "High Interest" / "Normal" / "Hide"
- **Filters**:
  - By importance (show only 4-5)
  - By topic
  - By feed
  - By date range
  - Read/unread status
  - Saved status (show only saved articles)
  - Archived status (show only archived articles)

### 5. Saved & Archived Articles

- **Save for Later**: Mark articles to read later
  - Individual save/unsave button on article cards
  - Bulk save/unsave operations with checkbox selection
  - Dedicated `/saved` page showing all saved articles
  - Filter articles by saved status in main dashboard
  - Per-user state (one user saving an article doesn't affect others)

- **Archive Articles**: Keep articles but remove from main view
  - Individual archive/unarchive button on article cards
  - Bulk archive/unarchive operations with checkbox selection
  - Dedicated `/archive` page showing all archived articles
  - Filter articles by archived status
  - Per-user state (archived articles are user-specific)

- **Use Cases**:
  - Save: "I want to read this carefully later"
  - Archive: "I've read this, keep it for reference but don't clutter my main view"
  - Read vs Saved vs Archived: Three independent states for organizing articles

### 6. User Interface

**Main Views:**
- **Dashboard**: Latest articles with summaries, sorted by importance
  - Shows processing status badges for articles still being summarized
  - Displays feed health warnings (error/warning status)
  - Save/archive buttons on each article card
  - Bulk selection mode for batch operations
- **Saved Articles** (`/saved`): All articles marked as saved
  - Dedicated view for articles to read later
  - Bulk unsave operations
  - Same filtering options as dashboard (importance, topic, feed)
- **Archived Articles** (`/archive`): All articles marked as archived
  - Keep articles for reference without cluttering main view
  - Bulk unarchive operations
  - Full search and filter capabilities
- **Feed Management**: List of feeds with CRUD operations
  - Shows feed status (healthy/warning/error) with error details
  - Edit button on each feed card opens modal for updating settings
  - Allows per-feed poll interval override (15-1440 minutes)
  - Toggle feed active/paused state
- **Article Detail**: Full summary, key points, topics, link to original
  - Shows processing errors if article failed
  - Retry button for failed articles
  - Save/archive/read toggle buttons
  - Previous/Next navigation buttons for unread articles (bottom of page)
  - Keyboard shortcuts: ← for previous, → for next
  - Position indicator (e.g., "5 of 23")
- **Topics Manager**: Manage broad topic categories with preference settings
  - Shows 15-20 broad topics only (e.g., "Go", "Security", "AI", "Kubernetes")
  - Topics fetched from normalized topics API (case-insensitive, deduplicated)
  - Only displays topics currently referenced by articles
  - Can set preferences per topic: "High Interest", "Normal", or "Hide"
  - Custom topics allowed but encouraged to follow broad category approach
- **Settings**: User preferences management
  - Default poll interval for all feeds (can be overridden per-feed)
  - LLM provider selection (OpenAI, Anthropic)
  - LLM model selection
  - **Custom LLM API Key**: Users can provide their own API key (stored encrypted with AES-256-GCM)
  - Max articles per feed per poll

## Data Models

### User
```go
type User struct {
    ID        string
    Email     string
    Name      string
    Provider  string // "google"
    CreatedAt time.Time
}
```

### UserPreferences
```go
type UserPreferences struct {
    ID                  string
    UserID              string
    DefaultPollInterval int    // minutes, default: 30
    LLMProvider         string // "openai" or "anthropic"
    LLMModel            string // e.g., "gpt-4", "claude-3-sonnet"
    LLMAPIKey           string // encrypted user's own API key (optional)
    MaxArticlesPerFeed  int    // limit articles fetched per poll, default: 20
    UpdatedAt           time.Time
}
```

### Feed
```go
type Feed struct {
    ID            string
    UserID        string
    URL           string
    Title         string
    Description   string
    PollFrequency int       // minutes, overrides user default if set
    LastPolledAt  time.Time
    IsActive      bool
    Status        string    // "healthy", "warning", "error", "paused"
    LastError     string    // last error message if status is error/warning
    ErrorCount    int       // consecutive errors, reset on success
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### Article
```go
// Global article shared across all users
type Article struct {
    ID               string
    FeedID           string
    Title            string
    URL              string
    PublishedAt      time.Time
    OriginalContent  string // RSS content
    FullText         string // scraped if needed
    Summary          string
    KeyPoints        []string
    ImportanceScore  int    // 1-5
    Topics           []string
    ProcessingStatus string // "pending", "processing", "completed", "failed"
    ProcessingError  string // error message if status is failed
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

### UserArticle
```go
// Per-user state for articles (read, saved, archived)
type UserArticle struct {
    ID         string
    UserID     string
    ArticleID  string
    IsRead     bool
    IsSaved    bool      // true = saved for later reading
    IsArchived bool      // true = archived (kept but hidden from main view)
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

**Note**: Articles are global entities shared across users. Each user's interaction state (read, saved, archived) is stored separately in the `user_articles` table. This allows multiple users to have different states for the same article.

### Topic
```go
// Global topic shared across all users
type Topic struct {
    ID       string
    Name     string
    IsCustom bool   // user-created vs auto-detected
}
```

### UserTopicPreference
```go
// Per-user topic preferences
type UserTopicPreference struct {
    ID         string
    UserID     string
    TopicID    string
    Preference string // "high", "normal", "hide"
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

**Note**: Topics are global entities shared across users (e.g., "Go", "Security", "AI"). Each user's preference for a topic is stored separately in the `user_topic_preferences` table. This prevents duplicate topics and allows efficient topic management.

**Database Indexes**:
- `user_articles`: Composite indexes on `(user_id, is_read)`, `(user_id, is_saved)`, `(user_id, is_archived)` for efficient filtering
- `topics`: Case-insensitive unique constraint on `LOWER(name)` to prevent duplicates like "AI", "ai", "Ai"
- `user_topic_preferences`: Composite unique constraint on `(user_id, topic_id)` to prevent duplicate preferences

## User Flows

### 1. Adding a New Feed
1. User clicks "Add Feed" button
2. User enters RSS feed URL
3. System validates and fetches feed metadata
4. System displays preview (title, description, recent items)
5. User confirms and feed is saved
6. Background workflow starts polling the feed

### 2. Editing a Feed
1. User clicks "Edit" button on a feed card in Feed Management view
2. Modal opens showing current feed settings:
   - Feed title (editable)
   - Poll frequency in minutes (15-1440 range)
   - Active status (checkbox to enable/disable polling)
3. User modifies desired settings:
   - Update feed title for easier identification
   - Adjust poll frequency (e.g., 30 minutes for important feeds, 120 for less critical)
   - Toggle active status to pause/resume polling without deleting feed
4. User clicks "Save Changes"
5. System validates inputs:
   - Title is not empty
   - Poll frequency is within valid range (15-1440 minutes)
6. Settings are updated immediately
7. Next poll will use new frequency if changed
8. Feed status indicator updates if active state was toggled

### 3. Viewing Summaries
1. User opens dashboard
2. System displays latest articles, sorted by importance (5 → 1), with pagination
3. User sees: title, summary, importance badge, topics, feed name, publish date, processing status, save/archive buttons
4. Articles still processing show "Processing..." badge instead of summary
5. Failed articles show "Failed" badge with retry option
6. User can filter by topic, importance, feed, processing status, saved status, or archived status
   - Topics dropdown shows normalized topics from API (not duplicates)
   - Filter to show only saved articles: useful for reading list
   - Filter to exclude archived articles: default behavior
7. User clicks article to view full summary or clicks "Read Original" to open source
8. User can save/archive articles individually or using bulk operations:
   - Click bookmark icon to save for later
   - Click archive icon to remove from main view
   - Enable bulk mode to select multiple articles
   - Perform bulk save, archive, or mark read operations
9. **Article Navigation**: From article detail page, user can navigate to next/previous unread article
   - Previous/Next buttons at bottom of article
   - Keyboard shortcuts: ← and → arrow keys
   - Shows position in unread queue (e.g., "5 of 23")
10. **Dedicated Views**:
    - Visit `/saved` to see all saved articles
    - Visit `/archive` to browse archived articles

### 4. Managing Preferences
1. User opens Settings page
2. User updates default poll interval (e.g., from 30 to 15 minutes)
3. User selects LLM provider (OpenAI or Anthropic) and model
4. **User can provide their own LLM API key** (optional)
   - API key is encrypted before storage using AES-256-GCM
   - If provided, system uses user's key instead of default
   - Allows users to control costs and avoid shared rate limits
5. User sets max articles per feed per poll
6. Changes are saved and take effect immediately
7. Existing feeds without custom poll intervals now use new default

### 5. Managing Topics
1. User opens Topics Manager
2. System displays 15-20 broad topics (e.g., "Go", "Security", "AI", "Kubernetes")
   - Each topic shows article count
   - Topics are auto-detected from articles via 3-layer normalization system
   - Only topics currently in use are shown (unused topics auto-deleted)
3. User sets preferences per topic:
   - "High Interest" - boosts importance rating for future articles
   - "Normal" - default rating behavior
   - "Hide" - filters out articles with this topic
4. User can create custom topics (encouraged to use broad categories)
5. Topic preferences affect future article importance ratings
6. Note: Topics are intentionally broad to prevent clutter and improve filtering

### 6. Saving and Archiving Articles
1. **Saving for Later**:
   - User sees an article of interest in the dashboard
   - User clicks the "Save" button (bookmark icon) on the article card
   - Article is marked as saved (saved status is per-user)
   - User can access all saved articles via the `/saved` page
   - User can unsave articles individually or use bulk operations

2. **Archiving Articles**:
   - User has read an article and wants to keep it but remove from main view
   - User clicks the "Archive" button on the article card
   - Article is archived (archived status is per-user)
   - Archived articles don't appear in main dashboard by default
   - User can access archived articles via the `/archive` page
   - User can unarchive articles to restore them to main view

3. **Bulk Operations**:
   - User enables "Bulk Mode" in article list
   - User selects multiple articles using checkboxes
   - User clicks "Save Selected", "Archive Selected", or "Mark Read"
   - All selected articles are updated in a single operation
   - Useful for quickly organizing many articles at once

4. **Filtering**:
   - In dashboard, user can filter to show only saved articles (`is_saved=true`)
   - User can filter to show only archived articles (`is_archived=true`)
   - Filters can be combined (e.g., saved + high importance + specific topic)

## API Endpoints

All endpoints are versioned starting with `/v1`.

### Authentication
- `POST /auth/google` - Initiate Google OAuth flow
- `POST /auth/callback` - OAuth callback handler
- `POST /auth/logout` - Invalidate session

### User Preferences
- `GET /v1/user/preferences` - Get current user's preferences
- `PUT /v1/user/preferences` - Update user preferences (poll interval, LLM provider, etc.)

### Feeds
- `GET /v1/feeds` - List all user feeds
  - Query params: `limit` (default: 50), `offset` (default: 0)
  - Response includes: `feeds[]`, `total_count`, `limit`, `offset`
- `POST /v1/feeds` - Add new feed
- `GET /v1/feeds/{id}` - Get feed details
- `PUT /v1/feeds/{id}` - Update feed (title, poll frequency, is_active, etc.)
- `DELETE /v1/feeds/{id}` - Delete feed
- `POST /v1/feeds/{id}/refresh` - Manual refresh trigger
- `GET /v1/feeds/{id}/health` - Get feed health status (recent errors, success rate)

### Articles
- `GET /v1/articles` - List articles
  - Query params: `limit` (default: 50), `offset` (default: 0), `min_importance` (min score), `topic`, `feed_id`, `is_read`, `is_saved`, `is_archived`, `processing_status`
  - Response includes: `articles[]`, `total_count`, `limit`, `offset`
- `GET /v1/articles/{id}` - Get article details
- `PATCH /v1/articles/{id}/read` - Mark as read/unread
  - Body: `{"is_read": true}`
- `PATCH /v1/articles/mark-read` - Bulk mark articles as read
  - Body: `{"article_ids": ["id1", "id2", ...], "is_read": true}`
- `PATCH /v1/articles/{id}/save` - Save/unsave article
  - Body: `{"is_saved": true}`
- `PATCH /v1/articles/bulk-save` - Bulk save/unsave articles
  - Body: `{"article_ids": ["id1", "id2", ...], "is_saved": true}`
- `PATCH /v1/articles/{id}/archive` - Archive/unarchive article
  - Body: `{"is_archived": true}`
- `PATCH /v1/articles/bulk-archive` - Bulk archive/unarchive articles
  - Body: `{"article_ids": ["id1", "id2", ...], "is_archived": true}`
- `POST /v1/articles/{id}/retry` - Retry failed article processing

### Topics
- `GET /v1/topics` - List all topics with user's preferences
  - Returns topics from user's feeds with their preference settings
  - Response includes: `topics[]` with `{id, name, preference, is_custom, article_count}`
- `PUT /v1/topics/{id}/preference` - Update user's preference for a topic
  - Body: `{"preference": "high" | "normal" | "hide"}`
- `POST /v1/topics` - Create custom topic
- `GET /v1/topics/{id}` - Get topic details
- `DELETE /v1/topics/{id}` - Delete custom topic (admin only)

## API Response Formats

### Paginated List Responses
All list endpoints (`GET /v1/feeds`, `GET /v1/articles`, etc.) return paginated responses:

```json
{
  "items": [...],
  "total_count": 150,
  "limit": 50,
  "offset": 0
}
```

### Error Responses
```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": {}
}
```

## Non-Functional Requirements

### Performance
- Feed polling should not block user interactions
- Dashboard should load < 2 seconds
- Article summarization should complete within 30 seconds

### Reliability
- Temporal workflows handle retries for failed LLM calls or network issues
- Failed feeds don't crash the system (log errors, mark feed as errored)

### Security
- OAuth tokens stored securely (encrypted at rest)
- API endpoints require authentication
- Input validation on all feed URLs (prevent SSRF attacks)
- **User LLM API keys encrypted** with AES-256-GCM using server encryption key
  - 12-byte random nonce per encryption (acts as salt)
  - Automatic encryption/decryption at repository layer
  - Server encryption key must be exactly 32 bytes (AES-256)

### Scalability
- Personal use: Handle 50-100 feeds, ~500 new articles/day per user
- Database indexes on common query patterns (user_id, importance, published_at)

## Success Metrics

- **User Engagement**: Daily active usage (user checks summaries)
- **Time Saved**: Average time to scan summaries vs reading full articles
- **Accuracy**: User feedback on importance ratings (thumbs up/down)
- **Feed Health**: % of feeds successfully polled without errors

## Additional Features (Post-MVP)

These features can be implemented after the core MVP is stable. They're listed in rough priority order based on user value.

### 1. Email Newsletter Support
Extend the platform to handle email newsletters the same way we handle RSS feeds.

**How it works:**
- User connects email account (Gmail, Outlook) via OAuth
- User creates filters/rules to identify newsletters (by sender, subject patterns)
- System periodically fetches new emails matching the filters
- Emails are processed through the same pipeline: extract text → summarize → rate → tag topics
- Newsletters appear alongside RSS articles in the unified dashboard

**Additional data models:**
- `EmailSource`: Connected email account with OAuth credentials
- `NewsletterFilter`: Rules for identifying newsletters (sender domain, subject patterns)
- Extend `Article` model with `source_type` field: "rss" or "email"

**API endpoints:**
- `POST /v1/email-sources` - Connect email account
- `GET /v1/email-sources` - List connected accounts
- `POST /v1/newsletter-filters` - Add newsletter filter
- `GET /v1/newsletter-filters` - List filters

**Technical considerations:**
- Use Gmail API or Microsoft Graph API for email access
- Store OAuth tokens securely with refresh capability
- Handle email parsing (plain text, HTML, multipart)
- Mark emails as read after processing to avoid re-processing
- Consider IMAP as alternative to API-based access

### 2. Email Digests
Daily/weekly email with top articles and summaries.

**Features:**
- Scheduled digest emails (daily at 7am, weekly on Monday)
- Configurable: which articles to include (importance >= 4, unread only, etc.)
- Beautiful HTML email template with article cards
- "Mark as read" links directly in email
- Digest frequency preferences per user

### 3. Mobile App
Native or PWA for on-the-go access.

**Features:**
- Push notifications for high-importance articles (score >= 4)
- Offline reading of summaries
- Swipe gestures for read/unread, save, archive
- Share articles to other apps
- Dark mode support

**Note**: Basic save/archive functionality is now in the MVP. Mobile app would provide touch-optimized UI for these features.

### 4. Browser Extension
One-click subscribe to RSS feeds while browsing.

**Features:**
- Auto-detect RSS/Atom feeds on current page
- One-click subscribe button in toolbar
- Right-click context menu to add current page's feed
- Badge showing unread count
- Quick view popup for latest summaries

### 5. Advanced Filtering & Views
More sophisticated filtering and organization.

**Features:**
- Saved filters/views (e.g., "AI articles this week", "must-read tech news")
- Reading goals (e.g., "read 10 articles per week")
- Full-text search across all articles and summaries
- **Advanced archive automation** (auto-archive articles after X days, auto-archive read articles)
- Folders/collections for organizing feeds
- Custom sorting options beyond importance

**Note**: Basic filtering (by importance, topic, feed, read/saved/archived status) and manual bulk archive operations are now in the MVP. This section covers advanced automation and saved filter presets.

### 6. Collaborative Features
Share and discuss summaries with others.

**Features:**
- Share individual summaries via link (public or private)
- Team workspaces for shared feeds and reading lists
- Comments/notes on articles
- Recommend articles to specific team members

### 7. Analytics Dashboard
Insights into reading patterns and content trends.

**Features:**
- Reading stats: articles read per day/week, time saved
- Topic trends: which topics are appearing more frequently
- Feed performance: which feeds produce the most high-value content
- Reading velocity: how quickly you're getting through articles
- Topic evolution: how your interests are changing over time

### 8. Multi-LLM Support
Choose between different AI providers.

**Features:**
- Support for OpenAI, Anthropic, Cohere, local models (Ollama)
- Per-feed LLM selection (use cheaper models for low-value feeds)
- Cost tracking and budgeting
- Fallback providers if primary fails
- A/B test summaries from different models

### 9. OPML Import/Export
Standard format for feed management.

**Features:**
- Import OPML file from other RSS readers
- Export feeds to OPML for backup or migration
- Bulk feed operations

### 10. Advanced Content Archiving & Export
Long-term storage and export of articles.

**Features:**
- Full-text search across archived content
- Export articles to PDF, Markdown, or Notion
- Permanent archive even if original source disappears
- Integration with read-it-later services (Pocket, Instapaper)
- Batch export of saved/archived articles
- Archive compression and storage optimization

**Note**: Basic archive functionality (marking articles as archived to hide from main view) is now in the MVP. This section covers advanced export, integration with external services, and storage optimization.
