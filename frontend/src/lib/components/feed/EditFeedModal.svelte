<script lang="ts">
	import type { FeedResponseBody } from '$lib/api/generated';
	import { feedStore } from '$lib/stores/feeds.svelte';

	type Feed = FeedResponseBody;

	let {
		feed,
		onClose
	}: {
		feed: Feed;
		onClose: () => void;
	} = $props();

	let title = $state('');
	let pollFrequencyMinutes = $state(60);
	let isActive = $state(true);
	let isSubmitting = $state(false);
	let error = $state<string | null>(null);

	// Initialize form fields from feed prop
	$effect(() => {
		title = feed.title;
		pollFrequencyMinutes = feed.poll_frequency_minutes;
		isActive = feed.is_active;
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		isSubmitting = true;
		error = null;

		try {
			await feedStore.updateFeed(feed.id, {
				title,
				poll_frequency_minutes: pollFrequencyMinutes,
				is_active: isActive
			});
			onClose();
		} catch (err: any) {
			error = err.message || 'Failed to update feed';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="bg-white rounded-lg shadow-xl p-6">
	<h2 class="text-xl font-bold text-gray-900 mb-4">Edit Feed</h2>

	<form onsubmit={handleSubmit} class="space-y-4">
		<!-- Title -->
		<div>
			<label for="title" class="block text-sm font-medium text-gray-700 mb-1"> Title </label>
			<input
				id="title"
				type="text"
				bind:value={title}
				required
				class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
			/>
		</div>

		<!-- Poll Frequency -->
		<div>
			<label for="poll-frequency" class="block text-sm font-medium text-gray-700 mb-1">
				Poll Frequency (minutes)
			</label>
			<input
				id="poll-frequency"
				type="number"
				bind:value={pollFrequencyMinutes}
				min="15"
				max="1440"
				required
				class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
			/>
			<p class="text-xs text-gray-500 mt-1">Between 15 minutes and 24 hours (1440 minutes)</p>
		</div>

		<!-- Is Active -->
		<div class="flex items-center">
			<input
				id="is-active"
				type="checkbox"
				bind:checked={isActive}
				class="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
			/>
			<label for="is-active" class="ml-2 text-sm text-gray-700">
				Active (automatically poll this feed)
			</label>
		</div>

		{#if error}
			<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
				{error}
			</div>
		{/if}

		<!-- Actions -->
		<div class="flex gap-3 pt-2">
			<button
				type="button"
				onclick={onClose}
				class="flex-1 px-4 py-2 text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 font-medium"
			>
				Cancel
			</button>
			<button
				type="submit"
				disabled={isSubmitting}
				class="flex-1 px-4 py-2 text-white bg-blue-600 rounded-lg hover:bg-blue-700 font-medium disabled:opacity-50 disabled:cursor-not-allowed"
			>
				{isSubmitting ? 'Saving...' : 'Save Changes'}
			</button>
		</div>
	</form>
</div>
