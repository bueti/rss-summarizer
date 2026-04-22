<script lang="ts">
	import type { FeedResponseBody } from '$lib/api/generated';
	import { feedStore } from '$lib/stores/feeds.svelte';
	import EditFeedModal from './EditFeedModal.svelte';

	type Feed = FeedResponseBody;

	let { feed }: { feed: Feed } = $props();

	let isEditDialogOpen = $state(false);
	let isRefreshing = $state(false);
	let isDeleting = $state(false);

	async function handleRefresh() {
		isRefreshing = true;
		try {
			await feedStore.refreshFeed(feed.id);
			alert('Feed refresh triggered. New articles will be fetched soon.');
		} catch (err: any) {
			alert(err.message || 'Failed to refresh feed');
		} finally {
			isRefreshing = false;
		}
	}

	async function handleDelete() {
		if (!confirm(`Delete "${feed.title}"? All articles from this feed will also be deleted.`)) {
			return;
		}

		isDeleting = true;
		try {
			await feedStore.deleteFeed(feed.id);
		} catch (err: any) {
			alert(err.message || 'Failed to delete feed');
		} finally {
			isDeleting = false;
		}
	}

	function getStatusColor(status: string) {
		switch (status) {
			case 'healthy':
				return 'bg-green-100 text-green-800';
			case 'warning':
				return 'bg-yellow-100 text-yellow-800';
			case 'error':
				return 'bg-red-100 text-red-800';
			default:
				return 'bg-gray-100 text-gray-800';
		}
	}

	function formatDate(dateString: string | null | undefined) {
		if (!dateString) return 'Never';
		return new Date(dateString).toLocaleString('en-US', {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
</script>

<div class="bg-white rounded-lg shadow-md p-6 hover:shadow-lg transition-shadow">
	<div class="flex flex-col h-full">
		<!-- Header -->
		<div class="flex items-start justify-between mb-4">
			<div class="flex-1 min-w-0">
				<h3 class="text-lg font-semibold text-gray-900 truncate">{feed.title}</h3>
				<a
					href={feed.url}
					target="_blank"
					rel="noopener noreferrer"
					class="text-sm text-blue-600 hover:underline truncate block"
				>
					{feed.url}
				</a>
			</div>

			<!-- Status Badge -->
			<span class="ml-2 px-2 py-1 text-xs font-medium rounded-full {getStatusColor(feed.status)}">
				{feed.status}
			</span>
		</div>

		<!-- Description -->
		{#if feed.description}
			<p class="text-sm text-gray-600 mb-4 line-clamp-2">{feed.description}</p>
		{/if}

		<!-- Stats -->
		<div class="text-sm text-gray-500 space-y-1 mb-4">
			<div class="flex justify-between">
				<span>Poll frequency:</span>
				<span class="font-medium text-gray-700">{feed.poll_frequency_minutes} min</span>
			</div>
			<div class="flex justify-between">
				<span>Last polled:</span>
				<span class="font-medium text-gray-700">{formatDate(feed.last_polled_at)}</span>
			</div>
			{#if feed.error_count > 0}
				<div class="flex justify-between text-red-600">
					<span>Error count:</span>
					<span class="font-medium">{feed.error_count}</span>
				</div>
			{/if}
		</div>

		<!-- Error Message -->
		{#if feed.last_error}
			<div class="bg-red-50 border border-red-200 rounded p-2 mb-4">
				<p class="text-xs text-red-800 truncate">{feed.last_error}</p>
			</div>
		{/if}

		<!-- Actions -->
		<div class="flex gap-2 mt-auto">
			<button
				onclick={() => (isEditDialogOpen = true)}
				class="flex-1 px-3 py-2 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50"
			>
				Edit
			</button>
			<button
				onclick={handleRefresh}
				disabled={isRefreshing}
				class="flex-1 px-3 py-2 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
			>
				{isRefreshing ? 'Refreshing...' : 'Refresh'}
			</button>
			<button
				onclick={handleDelete}
				disabled={isDeleting}
				class="px-3 py-2 text-sm font-medium text-red-600 border border-red-300 rounded hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed"
			>
				{isDeleting ? 'Deleting...' : 'Delete'}
			</button>
		</div>
	</div>
</div>

<!-- Edit Dialog -->
{#if isEditDialogOpen}
	<div
		class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50"
		role="presentation"
		onclick={(e) => {
			if (e.target === e.currentTarget) isEditDialogOpen = false;
		}}
		onkeydown={(e) => {
			if (e.key === 'Escape') isEditDialogOpen = false;
		}}
	>
		<div class="max-w-md w-full">
			<EditFeedModal {feed} onClose={() => (isEditDialogOpen = false)} />
		</div>
	</div>
{/if}

<style>
	.line-clamp-2 {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
