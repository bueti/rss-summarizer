<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { googleCallback } from '$lib/api/generated';

	let error = $state<string | null>(null);
	let isProcessing = $state(true);

	onMount(async () => {
		const code = $page.url.searchParams.get('code');
		const state = $page.url.searchParams.get('state');

		if (!code || !state) {
			error = 'Missing authorization code or state';
			isProcessing = false;
			return;
		}

		try {
			// Call backend callback endpoint
			const response = await googleCallback({ code, state });

			// Re-initialize auth store to get user info
			await authStore.initialize();

			// Redirect to dashboard
			if (response.status === 200) {
				goto(response.data.redirect_url || '/');
			}
		} catch (err: any) {
			error = err.message || 'Authentication failed';
			isProcessing = false;
		}
	});
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50">
	<div class="max-w-md w-full space-y-8 text-center">
		{#if isProcessing}
			<div>
				<div
					class="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"
				></div>
				<p class="mt-4 text-gray-600">Completing sign in...</p>
			</div>
		{:else if error}
			<div class="bg-red-50 border border-red-200 rounded-md p-4">
				<p class="text-sm text-red-800">{error}</p>
				<a href="/login" class="mt-4 inline-block text-blue-600 hover:text-blue-800">
					Return to login
				</a>
			</div>
		{/if}
	</div>
</div>
