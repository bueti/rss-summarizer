<script lang="ts">
	import { onMount } from 'svelte';
	import { articleStore } from '$lib/stores/articles.svelte';
	import { feedStore } from '$lib/stores/feeds.svelte';
	import ArticleList from '$lib/components/article/ArticleList.svelte';
	import Pagination from '$lib/components/ui/Pagination.svelte';

	let selectedArticles = $state<Set<string>>(new Set());
	let bulkMode = $state(false);

	onMount(async () => {
		// Load feeds first (needed for article cards)
		await feedStore.fetchFeeds();

		// Fetch saved articles
		await articleStore.fetchArticles({ is_saved: true, limit: 20, offset: 0 });
	});

	async function handlePageChange(page: number) {
		await articleStore.fetchPage(page);
		selectedArticles.clear();
		bulkMode = false;
	}

	function toggleBulkMode() {
		bulkMode = !bulkMode;
		if (!bulkMode) {
			selectedArticles.clear();
		}
	}

	function handleSelect(id: string, checked: boolean) {
		if (checked) {
			selectedArticles.add(id);
		} else {
			selectedArticles.delete(id);
		}
		selectedArticles = selectedArticles; // Trigger reactivity
	}

	async function handleBulkUnsave() {
		if (selectedArticles.size === 0) return;

		try {
			await articleStore.bulkSetSaved(Array.from(selectedArticles), false);
			selectedArticles.clear();
			bulkMode = false;
		} catch (err) {
			console.error('Failed to bulk unsave:', err);
		}
	}

	async function handleBulkMarkRead() {
		if (selectedArticles.size === 0) return;

		try {
			await articleStore.bulkMarkAsRead(Array.from(selectedArticles), true);
			selectedArticles.clear();
		} catch (err) {
			console.error('Failed to bulk mark as read:', err);
		}
	}
</script>

<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<div class="mb-6">
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-3xl font-bold text-gray-900">Saved Articles</h1>
				<p class="mt-1 text-sm text-gray-500">
					{articleStore.totalCount} saved {articleStore.totalCount === 1 ? 'article' : 'articles'}
				</p>
			</div>

			<div class="flex gap-2">
				{#if bulkMode && selectedArticles.size > 0}
					<button
						onclick={handleBulkMarkRead}
						class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
					>
						Mark {selectedArticles.size} Read
					</button>
					<button
						onclick={handleBulkUnsave}
						class="px-4 py-2 text-sm font-medium text-white bg-orange-600 rounded-md hover:bg-orange-700"
					>
						Unsave {selectedArticles.size}
					</button>
				{/if}
				<button
					onclick={toggleBulkMode}
					class="px-4 py-2 text-sm font-medium border border-gray-300 rounded-md hover:bg-gray-50"
				>
					{bulkMode ? 'Cancel' : 'Bulk Actions'}
				</button>
			</div>
		</div>
	</div>

	{#if articleStore.isLoading}
		<div class="flex justify-center py-12">
			<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
		</div>
	{:else if articleStore.error}
		<div class="bg-red-50 border border-red-200 rounded-lg p-4">
			<p class="text-red-800">{articleStore.error}</p>
		</div>
	{:else if articleStore.articles.length === 0}
		<div class="bg-white rounded-lg shadow-md p-12 text-center">
			<div class="text-gray-400 mb-4">
				<svg
					class="mx-auto h-12 w-12"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z"
					/>
				</svg>
			</div>
			<h3 class="text-lg font-medium text-gray-900 mb-2">No saved articles</h3>
			<p class="text-gray-500">
				Click the star icon on any article to save it for later reading.
			</p>
		</div>
	{:else}
		<ArticleList
			articles={articleStore.articles}
			showCheckboxes={bulkMode}
			{selectedArticles}
			onSelect={handleSelect}
		/>

		{#if articleStore.totalPages > 1}
			<div class="mt-6">
				<Pagination
					currentPage={articleStore.currentPage}
					totalPages={articleStore.totalPages}
					onPageChange={handlePageChange}
				/>
			</div>
		{/if}
	{/if}
</div>
