<script lang="ts">
	import { onMount } from 'svelte';
	import { feedStore } from '$lib/stores/feeds.svelte';
	import FeedCard from '$lib/components/feed/FeedCard.svelte';
	import AddFeedForm from '$lib/components/feed/AddFeedForm.svelte';
	import Pagination from '$lib/components/ui/Pagination.svelte';

	let isAddDialogOpen = $state(false);

	onMount(() => {
		feedStore.fetchFeeds();
	});

	async function handlePageChange(page: number) {
		await feedStore.fetchPage(page);
	}
</script>

<div class="space-y-6">
	<div class="flex justify-between items-center">
		<h1 class="text-3xl font-bold text-gray-900">Your Feeds</h1>
		<button
			onclick={() => (isAddDialogOpen = true)}
			class="px-4 py-2 text-white bg-blue-600 rounded-lg hover:bg-blue-700 font-medium"
		>
			Add Feed
		</button>
	</div>

	{#if feedStore.error}
		<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
			{feedStore.error}
		</div>
	{/if}

	{#if feedStore.isLoading}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each Array(6) as _}
				<div class="h-48 animate-pulse bg-gray-200 rounded-lg"></div>
			{/each}
		</div>
	{:else if feedStore.feeds.length === 0}
		<div class="bg-white rounded-lg shadow-md p-12 text-center">
			<p class="text-gray-500 mb-4">No feeds yet. Add your first RSS feed to get started!</p>
			<button
				onclick={() => (isAddDialogOpen = true)}
				class="px-4 py-2 text-white bg-blue-600 rounded hover:bg-blue-700"
			>
				Add Your First Feed
			</button>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each feedStore.feeds as feed (feed.id)}
				<FeedCard {feed} />
			{/each}
		</div>

		<!-- Pagination -->
		<Pagination
			currentPage={feedStore.currentPage}
			totalPages={feedStore.totalPages}
			onPageChange={handlePageChange}
		/>
	{/if}

	<!-- Simple Modal Dialog -->
	{#if isAddDialogOpen}
		<div
			class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50"
			role="presentation"
			onclick={(e) => {
				if (e.target === e.currentTarget) isAddDialogOpen = false;
			}}
			onkeydown={(e) => {
				if (e.key === 'Escape') isAddDialogOpen = false;
			}}
		>
			<div class="max-w-md w-full">
				<AddFeedForm onSuccess={() => (isAddDialogOpen = false)} />
			</div>
		</div>
	{/if}
</div>
