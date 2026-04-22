# Svelte 5 Frontend Development Guide

Quick reference for RSS Summarizer frontend development.

---

## Project Structure

```
frontend/src/
├── lib/
│   ├── api/
│   │   ├── client.ts         # Custom fetch wrapper (auth + error handling)
│   │   └── generated.ts      # Auto-generated from OpenAPI (DO NOT EDIT)
│   ├── components/
│   │   ├── ui/               # Button, Card, Input, Modal, Pagination
│   │   ├── feed/             # FeedList, FeedCard, FeedForm
│   │   └── article/          # ArticleCard, ArticleList, ArticleFilters
│   ├── stores/               # Global state (Svelte 5 runes)
│   │   ├── articles.svelte.ts
│   │   ├── feeds.svelte.ts
│   │   ├── topics.svelte.ts
│   │   └── auth.svelte.ts
│   ├── types/                # TypeScript types (non-generated)
│   └── utils/                # Helper functions
├── routes/                   # SvelteKit routes
│   ├── +layout.svelte        # App shell, navigation
│   ├── +page.svelte          # Dashboard (/)
│   ├── saved/+page.svelte    # Saved articles (/saved)
│   ├── archive/+page.svelte  # Archived articles (/archive)
│   ├── feeds/+page.svelte    # Feed management (/feeds)
│   ├── topics/+page.svelte   # Topic preferences (/topics)
│   └── settings/+page.svelte # User settings (/settings)
├── scripts/
│   └── generate-api-client.ts # Orval wrapper with retries
├── orval.config.ts            # API client generation config
└── package.json
```

---

## Quick Start

```bash
# Install dependencies
npm install

# Generate API client (requires backend running)
npm run generate:api

# Start dev server (auto-generates API client)
npm run dev

# Build for production
npm run build
```

---

## API Client Generation

### How It Works

The TypeScript API client is **auto-generated** from the backend's OpenAPI spec using [Orval](https://orval.dev/).

**Process:**
1. Backend exposes OpenAPI spec at `http://localhost:8080/openapi.json` (Huma framework)
2. Orval fetches the spec and generates TypeScript types + API functions
3. Generated code uses custom `fetch` wrapper for auth + error handling
4. Output: `src/lib/api/generated.ts` (DO NOT EDIT MANUALLY)

**Configuration:** `orval.config.ts`
```typescript
export default defineConfig({
  api: {
    input: {
      target: `${OPENAPI_URL}/openapi.json`,  // Fetch OpenAPI spec
    },
    output: {
      target: './src/lib/api/generated.ts',   // Generated file
      client: 'fetch',                         // Use fetch API
      override: {
        mutator: {
          path: './src/lib/api/client.ts',    // Custom fetch wrapper
          name: 'customFetch',
        },
      },
    },
  },
});
```

**Custom fetch wrapper** (`src/lib/api/client.ts`):
- Adds `Authorization: Bearer <token>` header
- Handles base URL from environment
- Parses error responses
- Used by all generated API functions

### Regenerate API Client

**Manual:**
```bash
npm run generate:api
```

**Automatic:**
- Runs before `npm run dev`
- Runs before `npm run build`
- Retries up to 5 times if backend not ready (useful in Docker)

**Script:** `scripts/generate-api-client.ts`
- Wraps `npx orval` with retry logic
- Falls back to existing types if backend unavailable (dev mode only)
- Fails build if backend unavailable (production)

### Using Generated API

**Generated types:**
```typescript
import type {
  ArticleResponse,
  CreateFeedBody,
  ListArticlesParams,
} from '$lib/api/generated';
```

**Generated functions:**
```typescript
import {
  listArticles,
  getArticle,
  createFeed,
  markArticleRead,
} from '$lib/api/generated';

// Usage
const articles = await listArticles({ min_importance: 4, limit: 20 });
const article = await getArticle(articleId);
await createFeed({ url: 'https://blog.golang.org/feed.atom' });
await markArticleRead(articleId, { is_read: true });
```

**All generated functions:**
- Accept proper TypeScript types (validated at compile-time)
- Use `customFetch` wrapper (auth + error handling)
- Return typed responses
- Throw on HTTP errors (handle in try/catch)

---

## State Management (Svelte 5 Runes)

### Store Pattern

**Class-based stores** with Svelte 5 runes (`$state`, `$derived`):

```typescript
// stores/articles.svelte.ts
class ArticleStore {
  articles = $state<ArticleResponse[]>([]);
  isLoading = $state(false);
  error = $state<string | null>(null);
  totalCount = $state(0);
  currentFilters = $state<ArticleFilters>({});

  async fetchArticles(filters?: ArticleFilters) {
    this.isLoading = true;
    this.error = null;
    this.currentFilters = filters || {};

    try {
      const response = await listArticles(filters);
      this.articles = response.articles;
      this.totalCount = response.total_count;
    } catch (err) {
      this.error = err.message;
      console.error('Failed to fetch articles:', err);
    } finally {
      this.isLoading = false;
    }
  }

  async toggleSaved(id: string, isSaved: boolean) {
    await setArticleSaved(id, { is_saved: isSaved });
    const article = this.articles.find(a => a.id === id);
    if (article) article.is_saved = isSaved;
  }
}

export const articleStore = new ArticleStore();
```

### Using Stores in Components

```svelte
<script lang="ts">
import { onMount } from 'svelte';
import { articleStore } from '$lib/stores/articles.svelte';

onMount(() => {
  articleStore.fetchArticles({ min_importance: 4 });
});
</script>

{#if articleStore.isLoading}
  <p>Loading...</p>
{:else if articleStore.error}
  <p>Error: {articleStore.error}</p>
{:else}
  <ul>
    {#each articleStore.articles as article}
      <li>{article.title}</li>
    {/each}
  </ul>
{/if}
```

**Key stores:**
- `articleStore` - Articles, filters, pagination
- `feedStore` - Feeds list
- `topicStore` - Topics + preferences
- `authStore` - User auth state (future)

---

## Component Patterns

### Svelte 5 Runes

**State:**
```svelte
let count = $state(0);
let user = $state({ name: 'John', age: 30 });
```

**Derived values:**
```svelte
let doubled = $derived(count * 2);
let isAdult = $derived(user.age >= 18);
```

**Props:**
```svelte
<script lang="ts">
let { article, onSave }: {
  article: ArticleResponse;
  onSave?: (id: string) => void
} = $props();
</script>
```

**Effects:**
```svelte
$effect(() => {
  console.log('Article changed:', article.id);
});
```

**DON'T use old Svelte syntax:**
```svelte
// ❌ OLD: $: doubled = count * 2
// ❌ OLD: export let article: Article
// ❌ OLD: $: console.log(article)
```

### Component Example

```svelte
<script lang="ts">
import type { ArticleResponse } from '$lib/api/generated';
import { articleStore } from '$lib/stores/articles.svelte';

let { article }: { article: ArticleResponse } = $props();
let isSaving = $state(false);

async function handleSave() {
  isSaving = true;
  try {
    await articleStore.toggleSaved(article.id, !article.is_saved);
  } catch (err) {
    console.error('Failed to save:', err);
  } finally {
    isSaving = false;
  }
}
</script>

<article class="card">
  <h2>{article.title}</h2>
  <p>{article.summary}</p>

  <button
    onclick={handleSave}
    disabled={isSaving}
    class="btn {article.is_saved ? 'btn-primary' : 'btn-secondary'}"
  >
    {article.is_saved ? 'Saved' : 'Save'}
  </button>
</article>
```

---

## Styling (Tailwind CSS)

**Utility-first approach:**
```svelte
<div class="flex items-center justify-between p-4 bg-white rounded-lg shadow-md">
  <h2 class="text-xl font-semibold text-gray-900">Title</h2>
  <button class="px-4 py-2 text-white bg-blue-600 rounded hover:bg-blue-700">
    Click
  </button>
</div>
```

**Responsive:**
```svelte
<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
  <!-- Content -->
</div>
```

**Conditional classes:**
```svelte
<button class="btn {isActive ? 'bg-blue-500' : 'bg-gray-500'}">
  Click
</button>
```

**DON'T:**
- Use excessive `@apply` in `<style>` blocks
- Use inline styles when Tailwind classes exist
- Use arbitrary values unnecessarily: `p-[17px]` (use `p-4` instead)

---

## Routing (SvelteKit)

**File-based routing:**
- `routes/+page.svelte` → `/`
- `routes/feeds/+page.svelte` → `/feeds`
- `routes/articles/[id]/+page.svelte` → `/articles/:id`

**Navigation:**
```svelte
<script>
import { goto } from '$app/navigation';
</script>

<a href="/feeds">Feeds</a>
<button onclick={() => goto('/saved')}>View Saved</button>
```

**Layout:**
- `routes/+layout.svelte` wraps all pages
- Contains nav, footer, etc.
- Shared across all routes

---

## Common Patterns

### Loading States

```svelte
{#if store.isLoading}
  <p>Loading...</p>
{:else if store.error}
  <p class="text-red-600">Error: {store.error}</p>
{:else}
  <!-- Content -->
{/if}
```

### Forms

```svelte
<script lang="ts">
let url = $state('');
let isSubmitting = $state(false);

async function handleSubmit(e: Event) {
  e.preventDefault();
  isSubmitting = true;

  try {
    await createFeed({ url, poll_frequency_minutes: 60 });
    url = ''; // Clear form
  } catch (err) {
    console.error('Failed to create feed:', err);
  } finally {
    isSubmitting = false;
  }
}
</script>

<form onsubmit={handleSubmit}>
  <input bind:value={url} type="url" required />
  <button type="submit" disabled={isSubmitting}>
    {isSubmitting ? 'Adding...' : 'Add Feed'}
  </button>
</form>
```

### Pagination

```svelte
<script lang="ts">
import Pagination from '$lib/components/ui/Pagination.svelte';

async function handlePageChange(page: number) {
  await articleStore.fetchPage(page);
}
</script>

<Pagination
  currentPage={1}
  totalPages={10}
  onPageChange={handlePageChange}
/>
```

---

## Development Workflow

### Backend Must Be Running

The frontend depends on the backend's OpenAPI spec to generate the API client.

**Start backend:**
```bash
cd backend
make docker-up    # Start Postgres + Temporal
make migrate-up   # Run migrations
make run          # Start API server (port 8080)
```

**Then start frontend:**
```bash
cd frontend
npm run dev       # Auto-generates API client, starts dev server (port 5173)
```

### Hot Reload

- Backend changes → Restart backend → Run `npm run generate:api` in frontend
- Frontend changes → Auto-reloads (Vite HMR)
- Component changes → Instant update
- Store changes → Instant update

---

## Common Commands

```bash
# Development
npm run dev              # Start dev server (with API gen)
npm run generate:api     # Regenerate API client only

# Build
npm run build            # Production build
npm run preview          # Preview production build

# Type checking
npm run check            # Run Svelte check (TypeScript validation)

# Linting
npm run lint             # (if configured)
```

---

## Key Files

**DO NOT EDIT:**
- `src/lib/api/generated.ts` - Auto-generated API client

**EDIT CAREFULLY:**
- `src/lib/api/client.ts` - Custom fetch wrapper (auth, error handling)
- `orval.config.ts` - API generation config

**FREQUENTLY EDITED:**
- `src/lib/stores/*.svelte.ts` - Global state
- `src/lib/components/**/*.svelte` - UI components
- `src/routes/**/*.svelte` - Pages

---

## Architecture Notes

### API Client Flow

1. Backend handler → OpenAPI spec (Huma auto-generates)
2. Orval fetches spec → Generates TypeScript types + functions
3. Generated functions use `customFetch` wrapper
4. Components/stores call generated functions
5. Responses are fully typed

**Benefits:**
- No manual API typing
- Compile-time validation
- Auto-complete in IDE
- Backend changes automatically reflected after regeneration

### State Management

- **Stores** hold global state (articles, feeds, topics)
- **Components** read from stores, trigger actions
- **Stores** call API functions, update state
- Svelte 5 runes make stores reactive

### Component Organization

- **UI components** (`lib/components/ui`) - Reusable primitives
- **Domain components** (`lib/components/article`, `feed`, etc.) - Feature-specific
- **Pages** (`routes`) - Route handlers, compose components
- Keep components small and focused
