<script lang="ts">
	import { z } from 'zod';
	import { feedStore } from '$lib/stores/feeds.svelte';

	let { onSuccess }: { onSuccess?: () => void } = $props();

	const feedSchema = z.object({
		url: z.string().url('Must be a valid URL'),
		poll_frequency_minutes: z.number().min(15).max(1440)
	});

	let formData = $state({
		url: '',
		poll_frequency_minutes: 60
	});

	let errors = $state<Record<string, string>>({});
	let isSubmitting = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		errors = {};
		isSubmitting = true;

		try {
			const validated = feedSchema.parse(formData);
			await feedStore.createFeed(validated.url, validated.poll_frequency_minutes);

			// Reset form
			formData = { url: '', poll_frequency_minutes: 60 };
			onSuccess?.();
		} catch (err: any) {
			if (err instanceof z.ZodError) {
				err.issues.forEach((error: z.ZodIssue) => {
					if (error.path[0]) {
						errors[error.path[0].toString()] = error.message;
					}
				});
			} else {
				errors.general = err.message || 'Failed to add feed';
			}
		} finally {
			isSubmitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit} class="space-y-4 bg-white p-6 rounded-lg">
	<h2 class="text-2xl font-bold text-gray-900">Add New Feed</h2>

	<div class="space-y-2">
		<label for="url" class="block text-sm font-medium text-gray-700">Feed URL</label>
		<input
			id="url"
			type="url"
			bind:value={formData.url}
			placeholder="https://example.com/feed.xml"
			class="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 {errors.url
				? 'border-red-500'
				: ''}"
		/>
		{#if errors.url}
			<p class="text-sm text-red-600">{errors.url}</p>
		{/if}
	</div>

	<div class="space-y-2">
		<label for="poll_frequency" class="block text-sm font-medium text-gray-700"
			>Poll Frequency (minutes)</label
		>
		<input
			id="poll_frequency"
			type="number"
			bind:value={formData.poll_frequency_minutes}
			min={15}
			max={1440}
			class="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 {errors.poll_frequency_minutes
				? 'border-red-500'
				: ''}"
		/>
		{#if errors.poll_frequency_minutes}
			<p class="text-sm text-red-600">{errors.poll_frequency_minutes}</p>
		{/if}
	</div>

	{#if errors.general}
		<div class="bg-red-50 border border-red-200 text-red-700 px-3 py-2 rounded text-sm">
			{errors.general}
		</div>
	{/if}

	<div class="flex justify-end space-x-2 pt-4">
		<button
			type="submit"
			disabled={isSubmitting}
			class="px-4 py-2 text-white bg-blue-600 rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
		>
			{isSubmitting ? 'Adding...' : 'Add Feed'}
		</button>
	</div>
</form>
