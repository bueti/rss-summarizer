<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { getLlmConfig, updateLlmConfig } from '$lib/api/generated';
	import type { LLMConfigResponse } from '$lib/api/generated';

	type LLMConfig = LLMConfigResponse;

	let config = $state<LLMConfig | null>(null);
	let isLoading = $state(true);
	let isSaving = $state(false);
	let error = $state<string | null>(null);
	let successMessage = $state<string | null>(null);

	let formData = $state({
		provider: 'anthropic',
		model: '',
		api_url: '',
		api_key: ''
	});

	onMount(async () => {
		// Check if user is admin
		if (!authStore.isAdmin) {
			goto('/');
			return;
		}

		await fetchConfig();
	});

	async function fetchConfig() {
		isLoading = true;
		error = null;

		try {
			const response = await getLlmConfig();
			if (response.status === 200) {
				config = response.data;
				formData.provider = response.data.provider || 'anthropic';
				formData.model = response.data.model || '';
				formData.api_url = response.data.api_url || '';
				formData.api_key = ''; // Never populate the API key field
			}
		} catch (err: any) {
			error = err.message || 'Failed to load LLM configuration';
			console.error('Error fetching LLM config:', err);
		} finally {
			isLoading = false;
		}
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		isSaving = true;
		error = null;
		successMessage = null;

		try {
			const body: any = {
				provider: formData.provider,
				model: formData.model,
				api_url: formData.api_url
			};

			// Only include API key if it was entered
			if (formData.api_key) {
				body.api_key = formData.api_key;
			}

			const response = await updateLlmConfig(body);
			if (response.status === 200) {
				config = response.data;
				formData.api_key = ''; // Clear API key field after successful update
				successMessage = 'LLM configuration updated successfully!';
				setTimeout(() => {
					successMessage = null;
				}, 3000);
			}
		} catch (err: any) {
			error = err.message || 'Failed to update LLM configuration';
			console.error('Error updating LLM config:', err);
		} finally {
			isSaving = false;
		}
	}
</script>

<svelte:head>
	<title>LLM Configuration - RSS Summarizer</title>
</svelte:head>

<div class="max-w-2xl mx-auto px-4 py-8">
	<h1 class="text-3xl font-bold text-gray-900 mb-8">LLM Configuration</h1>

	{#if isLoading}
		<div class="text-center py-12">
			<div
				class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-blue-600 border-r-transparent"
			></div>
			<p class="mt-2 text-gray-600">Loading configuration...</p>
		</div>
	{:else}
		<form onsubmit={handleSubmit} class="space-y-6 bg-white shadow-md rounded-lg p-6">
			<p class="text-sm text-gray-600 mb-4">
				Configure the global LLM provider and model used for article summarization. Changes apply
				to all users.
			</p>

			<!-- Provider -->
			<div>
				<label for="provider" class="block text-sm font-medium text-gray-700 mb-1">
					Provider
				</label>
				<select
					id="provider"
					bind:value={formData.provider}
					class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
				>
					<option value="openai">OpenAI</option>
					<option value="anthropic">Anthropic</option>
				</select>
				<p class="mt-1 text-sm text-gray-500">LLM provider to use</p>
			</div>

			<!-- Model -->
			<div>
				<label for="model" class="block text-sm font-medium text-gray-700 mb-1"> Model </label>
				<input
					id="model"
					type="text"
					bind:value={formData.model}
					placeholder="e.g., claude-3-5-sonnet-20241022 or gpt-4o-mini"
					class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
				/>
				<p class="mt-1 text-sm text-gray-500">Model name/identifier</p>
			</div>

			<!-- API URL -->
			<div>
				<label for="api_url" class="block text-sm font-medium text-gray-700 mb-1"> API URL </label>
				<input
					id="api_url"
					type="url"
					bind:value={formData.api_url}
					placeholder="e.g., https://api.anthropic.com/v1"
					class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
				/>
				<p class="mt-1 text-sm text-gray-500">API endpoint URL</p>
			</div>

			<!-- API Key -->
			<div>
				<label for="api_key" class="block text-sm font-medium text-gray-700 mb-1"> API Key </label>
				<input
					id="api_key"
					type="password"
					bind:value={formData.api_key}
					placeholder={config?.has_api_key ? '••••••••••••••••' : 'Enter API key'}
					class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
				/>
				<p class="mt-1 text-sm text-gray-500">
					{#if config?.has_api_key}
						Current: Set (leave blank to keep unchanged)
					{:else}
						Current: Not set
					{/if}
				</p>
			</div>

			<!-- Error Message -->
			{#if error}
				<div class="rounded-md bg-red-50 p-4">
					<p class="text-sm text-red-800">{error}</p>
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
					disabled={isSaving}
					class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{isSaving ? 'Saving...' : 'Save Configuration'}
				</button>
			</div>
		</form>
	{/if}
</div>
