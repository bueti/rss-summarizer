<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { topicsStore } from '$lib/stores/topics.svelte';
	import TopicCard from '$lib/components/topic/TopicCard.svelte';

	let showCreateModal = $state(false);
	let newTopicName = $state('');

	onMount(() => {
		topicsStore.fetch();
	});

	async function handleCreate(e: Event) {
		e.preventDefault();

		if (!newTopicName.trim()) return;

		const success = await topicsStore.create(newTopicName.trim());

		if (success) {
			newTopicName = '';
			showCreateModal = false;
		}
	}
</script>

<svelte:head>
	<title>Topics - RSS Summarizer</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 py-8">
	<div class="flex items-center justify-between mb-8">
		<h1 class="text-3xl font-bold text-gray-900">Topics</h1>
		<button
			onclick={() => (showCreateModal = true)}
			class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
		>
			Create Custom Topic
		</button>
	</div>

	{#if !browser}
		<div class="text-center py-12">
			<div
				class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-blue-600 border-r-transparent"
			></div>
			<p class="mt-2 text-gray-600">Loading topics...</p>
		</div>
	{:else if topicsStore.isLoading && topicsStore.topics.length === 0}
		<div class="text-center py-12">
			<div
				class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-blue-600 border-r-transparent"
			></div>
			<p class="mt-2 text-gray-600">Loading topics...</p>
		</div>
	{:else if topicsStore.error}
		<div class="rounded-md bg-red-50 p-4">
			<p class="text-sm text-red-800">{topicsStore.error}</p>
		</div>
	{:else if topicsStore.topics.length === 0}
		<div class="text-center py-12">
			<p class="text-gray-600">No topics yet. Topics will appear as articles are summarized.</p>
		</div>
	{:else}
		<div class="grid gap-4 md:grid-cols-2">
			{#each topicsStore.topics as topic (topic.id)}
				<TopicCard {topic} />
			{/each}
		</div>
	{/if}
</div>

<!-- Create Topic Modal -->
{#if showCreateModal}
	<div
		class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50"
		role="presentation"
		onclick={(e) => {
			if (e.target === e.currentTarget) showCreateModal = false;
		}}
		onkeydown={(e) => {
			if (e.key === 'Escape') showCreateModal = false;
		}}
	>
		<div class="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
			<h2 class="text-xl font-bold text-gray-900 mb-4">Create Custom Topic</h2>

			<form onsubmit={handleCreate} class="space-y-4">
				<div>
					<label for="topic-name" class="block text-sm font-medium text-gray-700 mb-1">
						Topic Name
					</label>
					<input
						id="topic-name"
						type="text"
						bind:value={newTopicName}
						required
						class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
						placeholder="e.g., Machine Learning"
					/>
				</div>

				<p class="text-sm text-gray-600">
					After creating the topic, you can set your preference for it.
				</p>

				{#if browser && topicsStore.error}
					<div class="rounded-md bg-red-50 p-3">
						<p class="text-sm text-red-800">{topicsStore.error}</p>
					</div>
				{/if}

				<div class="flex justify-end gap-3 mt-6">
					<button
						type="button"
						onclick={() => {
							showCreateModal = false;
							newTopicName = '';
						}}
						class="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={browser && topicsStore.isLoading}
						class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{browser && topicsStore.isLoading ? 'Creating...' : 'Create'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
