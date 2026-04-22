<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';

	let status = $derived($page.url.searchParams.get('status') || 'success');
	let message = $derived(
		$page.url.searchParams.get('message') || 'Gmail account connected successfully!'
	);

	onMount(() => {
		// Close the popup window after a brief delay
		setTimeout(() => {
			window.close();
		}, 1500);
	});
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50">
	<div class="bg-white rounded-lg shadow-lg p-8 max-w-md text-center">
		{#if status === 'success'}
			<div class="mb-4">
				<svg
					class="mx-auto h-16 w-16 text-green-500"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			</div>
			<h2 class="text-2xl font-bold text-gray-900 mb-2">Success!</h2>
		{:else}
			<div class="mb-4">
				<svg
					class="mx-auto h-16 w-16 text-red-500"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			</div>
			<h2 class="text-2xl font-bold text-gray-900 mb-2">Error</h2>
		{/if}

		<p class="text-gray-600 mb-4">{message}</p>
		<p class="text-sm text-gray-500">This window will close automatically...</p>

		<button
			onclick={() => window.close()}
			class="mt-4 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
		>
			Close Window
		</button>
	</div>
</div>
