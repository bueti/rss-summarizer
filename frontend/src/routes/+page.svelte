<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { articleStore } from '$lib/stores/articles.svelte';
	import { feedStore } from '$lib/stores/feeds.svelte';
	import { emailSourcesStore } from '$lib/stores/emailSources.svelte';
	import ArticleList from '$lib/components/article/ArticleList.svelte';
	import ArticleFilters from '$lib/components/article/ArticleFilters.svelte';
	import type { ArticleFilters as Filters } from '$lib/types';

	let currentFilters = $state<Filters>({});

	// Initialize filters from URL on mount
	onMount(async () => {
		// Parse filters from URL search params
		const params = $page.url.searchParams;
		const filters: Filters = {};

		const feedId = params.get('feed');
		if (feedId) filters.feed_id = feedId;

		const emailSourceId = params.get('email_source');
		if (emailSourceId) filters.email_source_id = emailSourceId;

		const minImportance = params.get('importance');
		if (minImportance) filters.min_importance = parseInt(minImportance, 10);

		const topic = params.get('topic');
		if (topic) filters.topic = topic;

		const sortBy = params.get('sort');
		if (sortBy === 'date' || sortBy === 'importance') filters.sort_by = sortBy;

		const isRead = params.get('read');
		if (isRead === 'true') filters.is_read = true;
		else if (isRead === 'false') filters.is_read = false;
		else filters.is_read = false; // Default to showing only unread articles

		// Exclude archived articles from dashboard by default
		filters.is_archived = false;

		currentFilters = filters;

		// Fetch data with URL filters
		await Promise.all([
			feedStore.fetchFeeds(),
			emailSourcesStore.fetchSources(),
			articleStore.fetchArticles(filters)
		]);
	});

	async function handleFilterChange(filters: Filters) {
		currentFilters = filters;

		// Update URL with new filters (without adding history entry)
		const params = new URLSearchParams();

		if (filters.feed_id) params.set('feed', filters.feed_id);
		if (filters.email_source_id) params.set('email_source', filters.email_source_id);
		if (filters.min_importance) params.set('importance', filters.min_importance.toString());
		if (filters.topic) params.set('topic', filters.topic);
		if (filters.sort_by) params.set('sort', filters.sort_by);
		if (filters.is_read !== undefined) params.set('read', filters.is_read.toString());

		const newUrl = params.toString() ? `?${params.toString()}` : '/';
		goto(newUrl, { replaceState: true, keepFocus: true, noScroll: true });

		// Fetch articles with new filters
		await articleStore.fetchArticles(filters);
	}
</script>

<div class="space-y-6">
	<div class="flex justify-between items-center">
		<div>
			<h1 class="text-3xl font-bold text-gray-900">Dashboard</h1>
			<p class="text-gray-600 mt-1">
				{articleStore.articles.length} articles
				{#if articleStore.unreadCount > 0}
					<span class="text-blue-600 font-medium">
						({articleStore.unreadCount} unread)
					</span>
				{/if}
			</p>
		</div>
	</div>

	<ArticleFilters filters={currentFilters} onFilterChange={handleFilterChange} />

	{#if articleStore.error}
		<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
			{articleStore.error}
		</div>
	{/if}

	<ArticleList />
</div>
