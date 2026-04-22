<script lang="ts">
	import { onMount } from 'svelte';
	import { emailSourcesStore } from '$lib/stores/emailSources.svelte';
	import { newsletterFiltersStore } from '$lib/stores/newsletterFilters.svelte';
	import EmailSourceCard from '$lib/components/email/EmailSourceCard.svelte';
	import NewsletterFilterForm from '$lib/components/email/NewsletterFilterForm.svelte';
	import NewsletterFiltersList from '$lib/components/email/NewsletterFiltersList.svelte';

	let isAddFilterDialogOpen = $state(false);

	onMount(() => {
		emailSourcesStore.fetchSources();
		newsletterFiltersStore.fetchFilters();
	});

	async function handleConnectGmail() {
		await emailSourcesStore.connectGmail();
	}
</script>

<div class="space-y-8">
	<!-- Email Sources Section -->
	<div class="space-y-6">
		<div class="flex justify-between items-center">
			<div>
				<h1 class="text-3xl font-bold text-gray-900">Email Sources</h1>
				<p class="text-gray-600 mt-1">Connect your Gmail account to receive newsletters</p>
			</div>
			<button
				onclick={handleConnectGmail}
				class="px-4 py-2 text-white bg-blue-600 rounded-lg hover:bg-blue-700 font-medium"
			>
				Connect Gmail
			</button>
		</div>

		{#if emailSourcesStore.error}
			<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
				{emailSourcesStore.error}
			</div>
		{/if}

		{#if emailSourcesStore.isLoading}
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				{#each Array(3) as _}
					<div class="h-48 animate-pulse bg-gray-200 rounded-lg"></div>
				{/each}
			</div>
		{:else if emailSourcesStore.sources.length === 0}
			<div class="bg-white rounded-lg shadow-md p-12 text-center">
				<p class="text-gray-500 mb-4">
					No email accounts connected. Connect your Gmail account to start receiving newsletters!
				</p>
				<button
					onclick={handleConnectGmail}
					class="px-4 py-2 text-white bg-blue-600 rounded hover:bg-blue-700"
				>
					Connect Gmail Account
				</button>
			</div>
		{:else}
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				{#each emailSourcesStore.sources as source (source.id)}
					<EmailSourceCard {source} />
				{/each}
			</div>
		{/if}
	</div>

	<!-- Newsletter Filters Section -->
	{#if emailSourcesStore.sources.length > 0}
		<div class="border-t pt-8 space-y-6">
			<div class="flex justify-between items-center">
				<div>
					<h2 class="text-2xl font-bold text-gray-900">Newsletter Filters</h2>
					<p class="text-gray-600 mt-1">
						Define which newsletters you want to receive and analyze
					</p>
				</div>
				<button
					onclick={() => (isAddFilterDialogOpen = true)}
					class="px-4 py-2 text-white bg-blue-600 rounded-lg hover:bg-blue-700 font-medium"
				>
					Add Filter
				</button>
			</div>

			{#if newsletterFiltersStore.error}
				<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
					{newsletterFiltersStore.error}
				</div>
			{/if}

			{#if newsletterFiltersStore.isLoading}
				<div class="space-y-3">
					{#each Array(3) as _}
						<div class="h-24 animate-pulse bg-gray-200 rounded-lg"></div>
					{/each}
				</div>
			{:else}
				<NewsletterFiltersList />
			{/if}

			<!-- Filter Examples -->
			<div class="bg-blue-50 border border-blue-200 rounded-lg p-4">
				<h3 class="text-sm font-semibold text-blue-900 mb-2">Filter Pattern Examples:</h3>
				<div class="text-xs text-blue-800 space-y-1">
					<div>
						<code class="bg-blue-100 px-2 py-0.5 rounded">*@substack.com</code> - All Substack
						newsletters
					</div>
					<div>
						<code class="bg-blue-100 px-2 py-0.5 rounded">newsletter@example.com</code> - Exact
						sender
					</div>
					<div>
						<code class="bg-blue-100 px-2 py-0.5 rounded">*newsletter*</code> - Contains "newsletter"
					</div>
					<div>
						Subject pattern (optional):
						<code class="bg-blue-100 px-2 py-0.5 rounded">^Weekly Digest</code> - Starts with
						"Weekly Digest"
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- Add Filter Dialog -->
	{#if isAddFilterDialogOpen}
		<div
			class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50"
			role="presentation"
			onclick={(e) => {
				if (e.target === e.currentTarget) isAddFilterDialogOpen = false;
			}}
			onkeydown={(e) => {
				if (e.key === 'Escape') isAddFilterDialogOpen = false;
			}}
		>
			<NewsletterFilterForm
				emailSources={emailSourcesStore.sources}
				onClose={() => (isAddFilterDialogOpen = false)}
			/>
		</div>
	{/if}
</div>
