<script lang="ts">
	import type { TopicWithPreference } from '$lib/api/generated';
	import { topicsStore } from '$lib/stores/topics.svelte';

	type TopicPreference = 'high' | 'normal' | 'hide';

	let { topic }: { topic: TopicWithPreference } = $props();

	const preferenceLabels: Record<string, string> = {
		high: 'High Interest',
		normal: 'Normal',
		hide: 'Hide'
	};

	const preferenceColors: Record<string, string> = {
		high: 'bg-blue-100 text-blue-800 border-blue-200',
		normal: 'bg-gray-100 text-gray-800 border-gray-200',
		hide: 'bg-red-100 text-red-800 border-red-200'
	};

	async function setPreference(newPreference: string) {
		await topicsStore.updatePreference(topic.id, newPreference as TopicPreference);
	}

	async function handleDelete() {
		if (confirm(`Delete topic "${topic.name}"?`)) {
			await topicsStore.delete(topic.id);
		}
	}
</script>

<div class="bg-white rounded-lg shadow p-4 border border-gray-200">
	<div class="flex items-center justify-between mb-3">
		<h3 class="text-lg font-semibold text-gray-900">{topic.name}</h3>
		{#if topic.is_custom}
			<button
				onclick={handleDelete}
				class="text-red-600 hover:text-red-800 text-sm font-medium"
				aria-label="Delete topic"
			>
				Delete
			</button>
		{/if}
	</div>

	<div class="flex items-center gap-2">
		<span class="text-sm text-gray-600 mr-2">Preference:</span>
		<div class="flex gap-2">
			{#each ['high', 'normal', 'hide'] as pref}
				<button
					onclick={() => setPreference(pref)}
					class="px-3 py-1 rounded-md text-sm font-medium transition-colors border {topic.preference ===
					pref
						? preferenceColors[pref]
						: 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'}"
				>
					{preferenceLabels[pref]}
				</button>
			{/each}
		</div>
	</div>

	{#if !topic.is_custom}
		<p class="mt-2 text-xs text-gray-500">Auto-detected topic</p>
	{/if}
</div>
