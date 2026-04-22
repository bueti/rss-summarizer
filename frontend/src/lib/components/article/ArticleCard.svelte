<script lang="ts">
	import type { ArticleResponse } from '$lib/api/generated';
	import { articleStore } from '$lib/stores/articles.svelte';
	import { feedStore } from '$lib/stores/feeds.svelte';
	import ProcessingStatusBadge from './ProcessingStatusBadge.svelte';

	type Article = ArticleResponse;

	let {
		article,
		showCheckbox = false,
		selected = false,
		onSelect
	}: {
		article: Article;
		showCheckbox?: boolean;
		selected?: boolean;
		onSelect?: (id: string, checked: boolean) => void;
	} = $props();

	const feed = $derived(article.feed_id ? feedStore.getFeedById(article.feed_id) : null);
	let isRetrying = $state(false);

	async function toggleRead() {
		try {
			await articleStore.markAsRead(article.id, !article.is_read);
		} catch (err) {
			console.error('Failed to toggle read status:', err);
		}
	}

	async function toggleSaved() {
		try {
			await articleStore.setSaved(article.id, !article.is_saved);
		} catch (err) {
			console.error('Failed to toggle saved status:', err);
		}
	}

	async function handleArchive() {
		try {
			await articleStore.setArchived(article.id, true);
		} catch (err) {
			console.error('Failed to archive article:', err);
		}
	}

	async function handleRetry() {
		isRetrying = true;
		try {
			await articleStore.retryArticle(article.id);

			// Optimistically update status to processing
			article.processing_status = 'processing';
			article.processing_error = undefined;
		} catch (err) {
			console.error('Failed to retry article:', err);
		} finally {
			isRetrying = false;
		}
	}

	function formatDate(dateString: string | undefined | null) {
		if (!dateString) return '';
		return new Date(dateString).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function getImportanceColor(score: number | undefined | null) {
		if (!score) return 'bg-gray-100 text-gray-800';
		if (score >= 5) return 'bg-red-100 text-red-800';
		if (score >= 4) return 'bg-orange-100 text-orange-800';
		if (score >= 3) return 'bg-blue-100 text-blue-800';
		return 'bg-gray-100 text-gray-800';
	}

	// Get the URL for viewing the original article/email
	const originalUrl = $derived(
		article.source_type === 'email' && article.email_message_id
			? `https://mail.google.com/mail/u/0/#inbox/${article.email_message_id}`
			: article.url
	);
</script>

<div
	class="bg-white rounded-lg shadow-md p-6 hover:shadow-lg transition-shadow border-l-4 {article.source_type ===
	'email'
		? 'border-purple-500'
		: 'border-blue-500'} {article.is_read ? 'opacity-75' : ''}"
>
	<div class="flex flex-col space-y-4">
		<!-- Header -->
		<div class="flex items-start gap-3">
			{#if showCheckbox}
				<input
					type="checkbox"
					checked={selected}
					onchange={(e) => onSelect?.(article.id, e.currentTarget.checked)}
					class="mt-1.5 h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
				/>
			{/if}

			<div class="flex-1 min-w-0">
				<a
					href="/articles/{article.id}"
					class="text-xl font-semibold text-gray-900 hover:text-blue-600 line-clamp-2 block"
				>
					{article.title}
				</a>
				<div class="flex items-center gap-2 mt-2 text-sm text-gray-500 flex-wrap">
					{#if article.source_type === 'email'}
						<span
							class="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-semibold bg-purple-100 text-purple-800 rounded-full"
						>
							<svg
								class="w-3.5 h-3.5"
								fill="currentColor"
								viewBox="0 0 20 20"
								xmlns="http://www.w3.org/2000/svg"
							>
								<path
									d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z"
								></path>
								<path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z"></path>
							</svg>
							Email Newsletter
						</span>
					{:else}
						<span
							class="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-semibold bg-blue-100 text-blue-800 rounded-full"
						>
							<svg
								class="w-3.5 h-3.5"
								fill="currentColor"
								viewBox="0 0 20 20"
								xmlns="http://www.w3.org/2000/svg"
							>
								<path
									d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM14 11a1 1 0 011 1v1h1a1 1 0 110 2h-1v1a1 1 0 11-2 0v-1h-1a1 1 0 110-2h1v-1a1 1 0 011-1z"
								></path>
							</svg>
							{feed?.title || 'RSS Feed'}
						</span>
					{/if}
					<span>•</span>
					<span>{formatDate(article.published_at)}</span>
					<span>•</span>
					<ProcessingStatusBadge status={article.processing_status} />
				</div>
			</div>

			{#if article.importance_score}
				<span
					class="ml-2 px-2 py-1 text-xs font-medium rounded-full {getImportanceColor(
						article.importance_score
					)}"
				>
					{article.importance_score}/5
				</span>
			{/if}
		</div>

		<!-- Processing Error -->
		{#if article.processing_error}
			<div class="bg-red-50 border border-red-200 rounded p-2">
				<p class="text-xs text-red-800">
					<span class="font-semibold">Processing Error:</span>
					{article.processing_error}
				</p>
			</div>
		{/if}

		<!-- Key Points -->
		{#if article.key_points && article.key_points.length > 0}
			<ul class="space-y-1.5 text-sm">
				{#each article.key_points.slice(0, 3) as point}
					<li class="flex items-start">
						<span class="text-blue-600 mr-2 font-semibold">•</span>
						<span class="line-clamp-1 text-gray-900">{point}</span>
					</li>
				{/each}
			</ul>
		{/if}

		<!-- Summary -->
		{#if article.summary}
			<p class="text-gray-600 text-sm line-clamp-2">{article.summary}</p>
		{/if}

		<!-- Topics -->
		{#if article.topics && article.topics.length > 0}
			<div class="flex flex-wrap gap-2">
				{#each article.topics as topic}
					<span class="px-2 py-1 text-xs font-medium bg-gray-100 text-gray-700 rounded-full"
						>{topic}</span
					>
				{/each}
			</div>
		{/if}

		<!-- Actions -->
		<div class="flex items-center justify-between pt-4 border-t">
			<div class="flex gap-2 flex-wrap">
				<button
					onclick={toggleRead}
					class="px-3 py-1.5 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50"
				>
					{article.is_read ? 'Mark Unread' : 'Mark Read'}
				</button>
				<button
					onclick={toggleSaved}
					class="px-3 py-1.5 text-sm font-medium border rounded {article.is_saved
						? 'bg-yellow-50 border-yellow-400 text-yellow-800 hover:bg-yellow-100'
						: 'border-gray-300 hover:bg-gray-50'}"
					title={article.is_saved ? 'Remove from saved' : 'Save for later'}
				>
					{article.is_saved ? '★ Saved' : '☆ Save'}
				</button>
				<button
					onclick={handleArchive}
					class="px-3 py-1.5 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50"
					title="Archive article"
				>
					Archive
				</button>
				<a
					href={originalUrl}
					target="_blank"
					rel="noopener noreferrer"
					class="px-3 py-1.5 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50"
				>
					View Original
				</a>
				{#if article.processing_status === 'failed' || article.processing_status === 'pending'}
					<button
						onclick={handleRetry}
						disabled={isRetrying}
						class="px-3 py-1.5 text-sm font-medium {article.processing_status === 'failed'
							? 'bg-orange-600 hover:bg-orange-700'
							: 'bg-blue-600 hover:bg-blue-700'} text-white rounded disabled:opacity-50 disabled:cursor-not-allowed"
						title={article.processing_status === 'failed' ? 'Retry processing' : 'Process now'}
					>
						{#if isRetrying}
							Processing...
						{:else if article.processing_status === 'failed'}
							↻ Retry
						{:else}
							▶ Process
						{/if}
					</button>
				{:else if article.processing_status === 'completed'}
					<button
						onclick={handleRetry}
						disabled={isRetrying}
						class="px-3 py-1.5 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
						title="Reprocess article (e.g., to regenerate summary with updated prompts)"
					>
						{#if isRetrying}
							Processing...
						{:else}
							↻ Reprocess
						{/if}
					</button>
				{/if}
			</div>

			<a
				href="/articles/{article.id}"
				class="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700"
			>
				Read More
			</a>
		</div>
	</div>
</div>

<style>
	.line-clamp-1 {
		display: -webkit-box;
		-webkit-line-clamp: 1;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.line-clamp-2 {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.line-clamp-3 {
		display: -webkit-box;
		-webkit-line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
