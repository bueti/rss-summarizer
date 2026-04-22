<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { feedStore } from '$lib/stores/feeds.svelte';
	import { articleStore } from '$lib/stores/articles.svelte';
	import type { ArticleResponse } from '$lib/api/generated';

	type Article = ArticleResponse;

	const articleId = $derived($page.params.id);

	let article = $state<Article | null>(null);
	let isLoading = $state(true);
	let isProcessing = $state(false);
	let error = $state<string | null>(null);
	let unreadArticles = $state<Article[]>([]);
	let currentIndex = $state(-1);

	const feed = $derived(article && article.feed_id ? feedStore.getFeedById(article.feed_id) : null);
	const hasPrevious = $derived(currentIndex > 0);
	const hasNext = $derived(currentIndex >= 0 && currentIndex < unreadArticles.length - 1);

	async function loadArticle(id: string) {
		isLoading = true;
		error = null;

		try {
			// Fetch current article
			article = await articleStore.getArticleById(id);

			// Update current index
			currentIndex = unreadArticles.findIndex((a) => a.id === id);

			// Mark as read automatically when viewing
			if (article && !article.is_read) {
				await articleStore.markAsRead(id, true);
				article.is_read = true;
			}
		} catch (err: any) {
			error = err.message || 'Failed to load article';
		} finally {
			isLoading = false;
		}
	}

	onMount(async () => {
		// Fetch unread articles list once
		unreadArticles = await articleStore.fetchUnreadArticles();
		// Load the current article
		if (articleId) {
			await loadArticle(articleId);
		}
	});

	// Reload article when URL changes
	$effect(() => {
		if (articleId && unreadArticles.length > 0) {
			loadArticle(articleId);
		}
	});

	async function handleProcess() {
		if (!article) return;

		isProcessing = true;
		try {
			await articleStore.retryArticle(article.id);

			// Update local status to show processing
			article.processing_status = 'processing';
		} catch (err: any) {
			alert(err.message || 'Failed to start processing');
			isProcessing = false;
		}
	}

	async function toggleRead() {
		if (!article) return;

		try {
			await articleStore.markAsRead(article.id, !article.is_read);
			article.is_read = !article.is_read;
		} catch (err) {
			alert('Failed to update read status');
		}
	}

	function formatDate(dateString: string | undefined) {
		if (!dateString) return 'Unknown date';
		return new Date(dateString).toLocaleDateString('en-US', {
			month: 'long',
			day: 'numeric',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function getImportanceColor(score: number | undefined) {
		if (!score) return 'bg-gray-100 text-gray-800';
		if (score >= 5) return 'bg-red-100 text-red-800';
		if (score >= 4) return 'bg-orange-100 text-orange-800';
		if (score >= 3) return 'bg-blue-100 text-blue-800';
		return 'bg-gray-100 text-gray-800';
	}

	function navigateToPrevious() {
		if (hasPrevious && currentIndex > 0) {
			const prevArticle = unreadArticles[currentIndex - 1];
			goto(`/articles/${prevArticle.id}`);
		}
	}

	function navigateToNext() {
		if (hasNext && currentIndex < unreadArticles.length - 1) {
			const nextArticle = unreadArticles[currentIndex + 1];
			goto(`/articles/${nextArticle.id}`);
		}
	}

	// Keyboard navigation
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'ArrowLeft' && hasPrevious) {
			navigateToPrevious();
		} else if (e.key === 'ArrowRight' && hasNext) {
			navigateToNext();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<svelte:head>
	<title>{article?.title || 'Article'} - RSS Summarizer</title>
</svelte:head>

{#if isLoading}
	<div class="bg-white rounded-lg shadow-md p-8">
		<div class="animate-pulse space-y-4">
			<div class="h-8 bg-gray-200 rounded w-3/4"></div>
			<div class="h-4 bg-gray-200 rounded w-1/2"></div>
			<div class="h-32 bg-gray-200 rounded"></div>
		</div>
	</div>
{:else if error}
	<div class="bg-white rounded-lg shadow-md p-8 text-center">
		<p class="text-red-600 mb-4">{error}</p>
		<a
			href="/"
			class="inline-block px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
		>
			Back to Dashboard
		</a>
	</div>
{:else if article}
	<div class="max-w-4xl mx-auto space-y-6">
		<!-- Header -->
		<div class="flex items-center justify-between">
			<a href="/" class="text-blue-600 hover:text-blue-700">← Back to Dashboard</a>
			<div class="flex gap-2">
				<button
					onclick={toggleRead}
					class="px-4 py-2 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50"
				>
					{article.is_read ? 'Mark Unread' : 'Mark Read'}
				</button>
				{#if article.processing_status === 'pending' || article.processing_status === 'failed'}
					<button
						onclick={handleProcess}
						disabled={isProcessing}
						class="px-4 py-2 text-sm font-medium {article.processing_status === 'failed'
							? 'bg-orange-600 hover:bg-orange-700'
							: 'bg-green-600 hover:bg-green-700'} text-white rounded disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{#if isProcessing}
							Processing...
						{:else if article.processing_status === 'failed'}
							↻ Retry Processing
						{:else}
							▶ Process Now
						{/if}
					</button>
				{:else if article.processing_status === 'completed'}
					<button
						onclick={handleProcess}
						disabled={isProcessing}
						class="px-4 py-2 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
						title="Reprocess article (e.g., to regenerate summary with updated prompts)"
					>
						{#if isProcessing}
							Processing...
						{:else}
							↻ Reprocess
						{/if}
					</button>
				{/if}
				<a
					href={article.url}
					target="_blank"
					rel="noopener noreferrer"
					class="px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700"
				>
					View Original →
				</a>
			</div>
		</div>

		<!-- Processing Indicator -->
		{#if article.processing_status === 'processing' || isProcessing}
			<div class="bg-blue-50 border-l-4 border-blue-500 p-4 rounded-r-lg">
				<div class="flex items-center">
					<div class="flex-shrink-0">
						<svg
							class="animate-spin h-5 w-5 text-blue-600"
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
						>
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
					</div>
					<div class="ml-3">
						<p class="text-sm font-medium text-blue-800">
							Processing article... This may take a minute.
						</p>
						<p class="text-xs text-blue-600 mt-1">Refresh the page to see the results.</p>
					</div>
				</div>
			</div>
		{/if}

		<!-- Article Content -->
		<div class="bg-white rounded-lg shadow-md p-8">
			<article class="prose prose-lg max-w-none">
				<!-- Title & Metadata -->
				<header class="mb-8">
					<h1 class="text-4xl font-bold text-gray-900 mb-4">{article.title}</h1>

					<div class="flex flex-wrap items-center gap-4 text-sm text-gray-600">
						<div class="flex items-center gap-2">
							<span class="font-medium">{feed?.title || 'Unknown Feed'}</span>
							<span>•</span>
							<span>{formatDate(article.published_at)}</span>
						</div>

						{#if article.importance_score}
							<span
								class="px-2 py-1 text-xs font-medium rounded-full {getImportanceColor(article.importance_score)}"
							>
								Importance: {article.importance_score}/5
							</span>
						{/if}

						{#if article.is_read}
							<span class="px-2 py-1 text-xs font-medium bg-gray-100 text-gray-700 rounded-full"
								>Read</span
							>
						{:else}
							<span class="px-2 py-1 text-xs font-medium bg-blue-100 text-blue-700 rounded-full"
								>Unread</span
							>
						{/if}
					</div>

					<!-- Topics -->
					{#if article.topics && article.topics.length > 0}
						<div class="flex flex-wrap gap-2 mt-4">
							{#each article.topics as topic}
								<span class="px-2 py-1 text-xs font-medium bg-gray-100 text-gray-700 rounded-full"
									>{topic}</span
								>
							{/each}
						</div>
					{/if}
				</header>

				<!-- Key Points -->
				{#if article.key_points && article.key_points.length > 0}
					<section class="bg-blue-50 border-l-4 border-blue-500 p-6 mb-8 rounded-r-lg">
						<h2 class="text-lg font-semibold text-gray-900 mb-3">Key Points</h2>
						<ul class="space-y-2">
							{#each article.key_points as point}
								<li class="flex items-start text-gray-700">
									<span class="text-blue-600 mr-2 font-bold">•</span>
									<span>{point}</span>
								</li>
							{/each}
						</ul>
					</section>
				{/if}

				<!-- Summary -->
				{#if article.summary}
					<section class="mb-8">
						<h2 class="text-2xl font-semibold text-gray-900 mb-4">Summary</h2>
						<div class="text-gray-700 whitespace-pre-line leading-relaxed">
							{article.summary}
						</div>
					</section>
				{/if}
			</article>
		</div>

		<!-- Navigation buttons at bottom -->
		{#if unreadArticles.length > 0}
			<div class="flex items-center justify-center gap-2 bg-white rounded-lg shadow-md p-4">
				<button
					onclick={navigateToPrevious}
					disabled={!hasPrevious}
					class="px-4 py-2 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
					title="Previous unread article (←)"
				>
					← Previous
				</button>
				<span class="text-sm text-gray-600 font-medium">
					{currentIndex + 1} of {unreadArticles.length}
				</span>
				<button
					onclick={navigateToNext}
					disabled={!hasNext}
					class="px-4 py-2 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
					title="Next unread article (→)"
				>
					Next →
				</button>
			</div>
		{/if}
	</div>
{/if}
