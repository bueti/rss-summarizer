<script lang="ts">
	import { onMount } from 'svelte';
	import { articleStore } from '$lib/stores/articles.svelte';
	import { feedStore } from '$lib/stores/feeds.svelte';
	import { emailSourcesStore } from '$lib/stores/emailSources.svelte';
	import { topicsStore } from '$lib/stores/topics.svelte';
	import type { ArticleFilters } from '$lib/types';

	let {
		filters,
		onFilterChange
	}: {
		filters: ArticleFilters;
		onFilterChange: (filters: ArticleFilters) => void;
	} = $props();

	let selectedFeedId = $state<string | undefined>(undefined);
	let selectedEmailSourceId = $state<string | undefined>(undefined);
	let selectedMinImportance = $state<number | undefined>(undefined);
	let selectedTopic = $state<string | undefined>(undefined);
	let selectedSortBy = $state<'date' | 'importance' | undefined>(undefined);
	let selectedIsRead = $state<string>('all');

	const allTopics = $derived(topicsStore.topicNames);
	const feeds = $derived(feedStore.feeds);
	const emailSources = $derived(emailSourcesStore.sources);

	// Sync local state when filters prop changes (from URL params)
	$effect(() => {
		selectedFeedId = filters.feed_id;
		selectedEmailSourceId = filters.email_source_id;
		selectedMinImportance = filters.min_importance;
		selectedTopic = filters.topic;
		selectedSortBy = filters.sort_by;
		selectedIsRead = filters.is_read === true ? 'true' : filters.is_read === false ? 'false' : 'all';
	});

	onMount(() => {
		topicsStore.fetchTopics();
		emailSourcesStore.fetchSources();
	});

	function applyFilters() {
		const newFilters: ArticleFilters = {
			feed_id: selectedFeedId || undefined,
			email_source_id: selectedEmailSourceId || undefined,
			min_importance: selectedMinImportance || undefined,
			topic: selectedTopic || undefined,
			sort_by: selectedSortBy || undefined,
			is_read:
				selectedIsRead === 'true' ? true : selectedIsRead === 'false' ? false : undefined
		};
		onFilterChange(newFilters);
	}

	function clearFilters() {
		selectedFeedId = undefined;
		selectedEmailSourceId = undefined;
		selectedMinImportance = undefined;
		selectedTopic = undefined;
		selectedSortBy = undefined;
		selectedIsRead = 'false'; // Default to showing unread articles
		isExpanded = false; // Collapse the menu
		onFilterChange({ is_read: false });
	}

	const hasActiveFilters = $derived(
		selectedFeedId || selectedEmailSourceId || selectedMinImportance || selectedTopic || selectedSortBy || (selectedIsRead !== 'false' && selectedIsRead !== 'all')
	);

	const activeFilterCount = $derived(
		[selectedFeedId, selectedEmailSourceId, selectedMinImportance, selectedTopic, selectedSortBy, (selectedIsRead !== 'false' && selectedIsRead !== 'all') ? true : undefined].filter(Boolean).length
	);

	// Expand by default if there are active filters, otherwise collapse
	let isExpanded = $state(false);

	// Auto-expand when active filters are present
	$effect(() => {
		if (hasActiveFilters) {
			isExpanded = true;
		}
	});

	function toggleExpanded() {
		isExpanded = !isExpanded;
	}
</script>

<div class="bg-white rounded-lg shadow-md p-4">
	<div class="flex items-center justify-between">
		<button
			onclick={toggleExpanded}
			class="flex items-center gap-2 text-sm font-semibold text-gray-700 hover:text-gray-900"
		>
			<svg
				class="w-4 h-4 transition-transform {isExpanded ? 'rotate-90' : ''}"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
			</svg>
			Filters
			{#if activeFilterCount > 0}
				<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
					{activeFilterCount} active
				</span>
			{/if}
		</button>
		{#if hasActiveFilters}
			<button
				onclick={clearFilters}
				class="text-sm text-blue-600 hover:text-blue-700 font-medium"
			>
				Clear all
			</button>
		{/if}
	</div>

	{#if isExpanded}
	<div class="flex flex-wrap gap-4 items-end mt-3">
		<div class="flex-1 min-w-[200px]">
			<label for="filter-feed" class="block text-sm font-medium text-gray-700 mb-1">RSS Feed</label>
			<select
				id="filter-feed"
				bind:value={selectedFeedId}
				onchange={applyFilters}
				class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
			>
				<option value={undefined}>All RSS Feeds</option>
				{#each feeds as feed}
					<option value={feed.id}>{feed.title}</option>
				{/each}
			</select>
		</div>

		<div class="flex-1 min-w-[200px]">
			<label for="filter-email-source" class="block text-sm font-medium text-gray-700 mb-1">Email Source</label>
			<select
				id="filter-email-source"
				bind:value={selectedEmailSourceId}
				onchange={applyFilters}
				class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-purple-500"
			>
				<option value={undefined}>All Email Sources</option>
				{#each emailSources as source}
					<option value={source.id}>{source.email_address}</option>
				{/each}
			</select>
		</div>

		<div class="flex-1 min-w-[200px]">
			<label for="filter-importance" class="block text-sm font-medium text-gray-700 mb-1"
				>Min Importance</label
			>
			<select
				id="filter-importance"
				bind:value={selectedMinImportance}
				onchange={applyFilters}
				class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
			>
				<option value={undefined}>Any</option>
				<option value={1}>1+</option>
				<option value={2}>2+</option>
				<option value={3}>3+</option>
				<option value={4}>4+</option>
				<option value={5}>5</option>
			</select>
		</div>

		<div class="flex-1 min-w-[200px]">
			<label for="filter-topic" class="block text-sm font-medium text-gray-700 mb-1">Topic</label>
			<select
				id="filter-topic"
				bind:value={selectedTopic}
				onchange={applyFilters}
				class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
			>
				<option value={undefined}>All Topics</option>
				{#each allTopics as topic}
					<option value={topic}>{topic}</option>
				{/each}
			</select>
		</div>

		<div class="flex-1 min-w-[200px]">
			<label for="filter-sort" class="block text-sm font-medium text-gray-700 mb-1">Sort By</label>
			<select
				id="filter-sort"
				bind:value={selectedSortBy}
				onchange={applyFilters}
				class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
			>
				<option value={undefined}>Date (Newest First)</option>
				<option value="importance">Importance (Highest First)</option>
			</select>
		</div>

		<div class="flex-1 min-w-[200px]">
			<label for="filter-read-status" class="block text-sm font-medium text-gray-700 mb-1"
				>Read Status</label
			>
			<select
				id="filter-read-status"
				bind:value={selectedIsRead}
				onchange={applyFilters}
				class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
			>
				<option value="all">All</option>
				<option value="false">Unread Only</option>
				<option value="true">Read Only</option>
			</select>
		</div>
	</div>
	{/if}
</div>
