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

		// Fetch archived articles
		await articleStore.fetchArticles({ is_archived: true, limit: 20, offset: 0 });
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

	async function handleBulkUnarchive() {
		if (selectedArticles.size === 0) return;

		try {
			await articleStore.bulkSetArchived(Array.from(selectedArticles), false);
			selectedArticles.clear();
			bulkMode = false;
		} catch (err) {
			console.error('Failed to bulk unarchive:', err);
		}
	}

	async function handleBulkSave() {
		if (selectedArticles.size === 0) return;

		try {
			await articleStore.bulkSetSaved(Array.from(selectedArticles), true);
			selectedArticles.clear();
		} catch (err) {
			console.error('Failed to bulk save:', err);
		}
	}
</script>

<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<div class="mb-6">
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-3xl font-bold text-gray-900">Archive</h1>
				<p class="mt-1 text-sm text-gray-500">
					{articleStore.totalCount} archived {articleStore.totalCount === 1
						? 'article'
						: 'articles'}
				</p>
			</div>

			<div class="flex gap-2">
				{#if bulkMode && selectedArticles.size > 0}
					<button
						onclick={handleBulkSave}
						class="px-4 py-2 text-sm font-medium text-white bg-yellow-600 rounded-md hover:bg-yellow-700"
					>
						Save {selectedArticles.size}
					</button>
					<button
						onclick={handleBulkUnarchive}
						class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
					>
						Unarchive {selectedArticles.size}
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
						d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
					/>
				</svg>
			</div>
			<h3 class="text-lg font-medium text-gray-900 mb-2">No archived articles</h3>
			<p class="text-gray-500">
				Articles you've read (and not saved) will automatically appear here.
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
