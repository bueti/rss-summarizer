<script lang="ts">
	import type { ArticleResponse } from '$lib/api/generated';
	import { articleStore } from '$lib/stores/articles.svelte';
	import ArticleCard from './ArticleCard.svelte';
	import Pagination from '$lib/components/ui/Pagination.svelte';

	type Article = ArticleResponse;

	// Accept optional props for custom usage (like archive/saved pages)
	let {
		articles: customArticles,
		showCheckboxes,
		selectedArticles,
		onSelect
	}: {
		articles?: Article[];
		showCheckboxes?: boolean;
		selectedArticles?: Set<string>;
		onSelect?: (id: string, checked: boolean) => void;
	} = $props();

	// Use custom articles if provided, otherwise use store articles
	const articles = $derived(customArticles ?? articleStore.articles);
	const isLoading = $derived(articleStore.isLoading);

	let bulkMode = $state(false);
	let selectedIds = $state<Set<string>>(new Set());
	let isProcessingBulk = $state(false);

	// Sync selectedIds with selectedArticles prop
	$effect(() => {
		selectedIds = selectedArticles ?? new Set();
	});

	// Sync bulkMode with showCheckboxes prop
	$effect(() => {
		bulkMode = showCheckboxes ?? false;
	});

	function handleSelect(id: string, checked: boolean) {
		// Call custom onSelect handler if provided
		if (onSelect) {
			onSelect(id, checked);
		} else {
			// Default behavior
			if (checked) {
				selectedIds.add(id);
			} else {
				selectedIds.delete(id);
			}
			selectedIds = new Set(selectedIds); // Trigger reactivity
		}
	}

	function toggleAll() {
		if (selectedIds.size === articles.length) {
			selectedIds.clear();
		} else {
			selectedIds = new Set(articles.map((a) => a.id));
		}
	}

	async function handleBulkMarkRead(isRead: boolean) {
		if (selectedIds.size === 0) return;

		isProcessingBulk = true;
		try {
			await articleStore.bulkMarkAsRead(Array.from(selectedIds), isRead);
			selectedIds.clear();
			bulkMode = false;
		} catch (err) {
			console.error('Failed to bulk mark articles:', err);
		} finally {
			isProcessingBulk = false;
		}
	}

	function cancelBulk() {
		bulkMode = false;
		selectedIds.clear();
	}

	async function handlePageChange(page: number) {
		await articleStore.fetchPage(page);
		// Reset bulk selection when page changes
		bulkMode = false;
		selectedIds.clear();
	}
</script>

<div class="space-y-4">
	<!-- Bulk Actions Header -->
	{#if articles.length > 0}
		<div class="flex items-center justify-between bg-white p-4 rounded-lg shadow-md">
			<div class="flex items-center gap-4">
				<button
					onclick={() => {
						bulkMode = !bulkMode;
						if (!bulkMode) selectedIds.clear();
					}}
					class="px-3 py-1.5 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50"
				>
					{bulkMode ? 'Cancel Selection' : 'Bulk Actions'}
				</button>

				{#if bulkMode}
					<button
						onclick={toggleAll}
						class="text-sm text-blue-600 hover:text-blue-800"
					>
						{selectedIds.size === articles.length ? 'Deselect All' : 'Select All'}
					</button>
					<span class="text-sm text-gray-600">{selectedIds.size} selected</span>
				{/if}
			</div>

			{#if bulkMode && selectedIds.size > 0}
				<div class="flex gap-2">
					<button
						onclick={() => handleBulkMarkRead(true)}
						disabled={isProcessingBulk}
						class="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
					>
						Mark Read
					</button>
					<button
						onclick={() => handleBulkMarkRead(false)}
						disabled={isProcessingBulk}
						class="px-3 py-1.5 text-sm font-medium bg-gray-600 text-white rounded hover:bg-gray-700 disabled:opacity-50"
					>
						Mark Unread
					</button>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Articles List -->
	{#if isLoading}
		<div class="space-y-4">
			{#each Array(5) as _}
				<div class="h-48 animate-pulse bg-gray-200 rounded-lg"></div>
			{/each}
		</div>
	{:else if articles.length === 0}
		<div class="bg-white rounded-lg shadow-md p-12 text-center">
			<p class="text-gray-500">No articles found. Try adjusting your filters or add more feeds.</p>
		</div>
	{:else}
		<div class="space-y-4">
			{#each articles as article (article.id)}
				<ArticleCard
					{article}
					showCheckbox={bulkMode}
					selected={selectedIds.has(article.id)}
					onSelect={handleSelect}
				/>
			{/each}
		</div>

		<!-- Pagination -->
		<Pagination
			currentPage={articleStore.currentPage}
			totalPages={articleStore.totalPages}
			onPageChange={handlePageChange}
		/>
	{/if}
</div>
