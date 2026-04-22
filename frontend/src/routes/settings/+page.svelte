<script lang="ts">
	import { onMount } from 'svelte';
	import { preferencesStore } from '$lib/stores/preferences.svelte';
	import type { UpdatePreferencesRequestBody } from '$lib/api/generated';

	let formData = $state<UpdatePreferencesRequestBody>({
		default_poll_interval: 60,
		max_articles_per_feed: 20
	});

	let successMessage = $state('');
	let validationErrors = $state<Record<string, string>>({});

	onMount(() => {
		preferencesStore.fetch();
	});

	// Update form when preferences load
	$effect(() => {
		if (preferencesStore.preferences) {
			formData.default_poll_interval = preferencesStore.preferences.default_poll_interval;
			formData.max_articles_per_feed = preferencesStore.preferences.max_articles_per_feed;
		}
	});

	function validate(): boolean {
		validationErrors = {};

		if (formData.default_poll_interval < 15 || formData.default_poll_interval > 1440) {
			validationErrors.default_poll_interval =
				'Poll interval must be between 15 and 1440 minutes';
		}

		if (formData.max_articles_per_feed < 1 || formData.max_articles_per_feed > 100) {
			validationErrors.max_articles_per_feed =
				'Max articles per feed must be between 1 and 100';
		}

		return Object.keys(validationErrors).length === 0;
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		successMessage = '';

		if (!validate()) {
			return;
		}

		const success = await preferencesStore.update(formData);

		if (success) {
			successMessage = 'Settings saved successfully!';
			setTimeout(() => {
				successMessage = '';
			}, 3000);
		}
	}
</script>

<svelte:head>
	<title>Settings - RSS Summarizer</title>
</svelte:head>

<div class="max-w-2xl mx-auto px-4 py-8">
	<h1 class="text-3xl font-bold text-gray-900 mb-8">Settings</h1>

	{#if preferencesStore.isLoading && !preferencesStore.preferences}
		<div class="text-center py-8">
			<div class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-blue-600 border-r-transparent"></div>
			<p class="mt-2 text-gray-600">Loading settings...</p>
		</div>
	{:else}
		<form onsubmit={handleSubmit} class="space-y-6 bg-white shadow-md rounded-lg p-6">
			<p class="text-sm text-gray-600 mb-4">
				Configure your personal RSS feed preferences. LLM configuration is managed globally by administrators.
			</p>

			<!-- Default Poll Interval -->
			<div>
				<label for="poll_interval" class="block text-sm font-medium text-gray-700 mb-1">
					Default Poll Interval (minutes)
				</label>
				<input
					id="poll_interval"
					type="number"
					bind:value={formData.default_poll_interval}
					min="15"
					max="1440"
					class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
				/>
				<p class="mt-1 text-sm text-gray-500">How often to check feeds (15-1440 minutes)</p>
				{#if validationErrors.default_poll_interval}
					<p class="mt-1 text-sm text-red-600">{validationErrors.default_poll_interval}</p>
				{/if}
			</div>

			<!-- Max Articles Per Feed -->
			<div>
				<label for="max_articles" class="block text-sm font-medium text-gray-700 mb-1">
					Max Articles Per Feed
				</label>
				<input
					id="max_articles"
					type="number"
					bind:value={formData.max_articles_per_feed}
					min="1"
					max="100"
					class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
				/>
				<p class="mt-1 text-sm text-gray-500">Maximum articles to fetch per feed per poll (1-100)</p>
				{#if validationErrors.max_articles_per_feed}
					<p class="mt-1 text-sm text-red-600">{validationErrors.max_articles_per_feed}</p>
				{/if}
			</div>

			<!-- Error Message -->
			{#if preferencesStore.error}
				<div class="rounded-md bg-red-50 p-4">
					<p class="text-sm text-red-800">{preferencesStore.error}</p>
				</div>
			{/if}

			<!-- Success Message -->
			{#if successMessage}
				<div class="rounded-md bg-green-50 p-4">
					<p class="text-sm text-green-800">{successMessage}</p>
				</div>
			{/if}

			<!-- Submit Button -->
			<div class="flex justify-end">
				<button
					type="submit"
					disabled={preferencesStore.isLoading}
					class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{preferencesStore.isLoading ? 'Saving...' : 'Save Settings'}
				</button>
			</div>
		</form>
	{/if}
</div>
