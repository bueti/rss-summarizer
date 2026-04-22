<script lang="ts">
	import type { EmailSource } from '$lib/api/generated';
	import { emailSourcesStore } from '$lib/stores/emailSources.svelte';

	let { source }: { source: EmailSource } = $props();

	let isDisconnecting = $state(false);

	async function handleDisconnect() {
		if (
			!confirm(
				`Disconnect Gmail account "${source.email_address}"? All newsletter filters for this account will also be deleted.`
			)
		) {
			return;
		}

		isDisconnecting = true;
		try {
			await emailSourcesStore.disconnect(source.id);
		} catch (err: any) {
			alert(err.message || 'Failed to disconnect email source');
		} finally {
			isDisconnecting = false;
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

	function getStatusColor(isActive: boolean) {
		return isActive ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800';
	}
</script>

<div class="bg-white rounded-lg shadow-md p-6 hover:shadow-lg transition-shadow">
	<div class="flex flex-col h-full">
		<!-- Header -->
		<div class="flex items-start justify-between mb-4">
			<div class="flex-1 min-w-0">
				<h3 class="text-lg font-semibold text-gray-900 truncate">{source.email_address}</h3>
				<p class="text-sm text-gray-500">Provider: {source.provider}</p>
			</div>

			<!-- Status Badge -->
			<span
				class="ml-2 px-2 py-1 text-xs font-medium rounded-full {getStatusColor(source.is_active)}"
			>
				{source.is_active ? 'Active' : 'Inactive'}
			</span>
		</div>

		<!-- Stats -->
		<div class="text-sm text-gray-500 space-y-1 mb-4">
			<div class="flex justify-between">
				<span>Last fetched:</span>
				<span class="font-medium text-gray-700">{formatDate(source.last_fetched_at)}</span>
			</div>
			<div class="flex justify-between">
				<span>Token expires:</span>
				<span class="font-medium text-gray-700">{formatDate(source.token_expires_at)}</span>
			</div>
		</div>

		<!-- Error Message -->
		{#if source.last_error}
			<div class="bg-red-50 border border-red-200 rounded p-2 mb-4">
				<p class="text-xs text-red-800 truncate">{source.last_error}</p>
				{#if !source.is_active}
					<p class="text-xs text-red-600 mt-1 font-medium">
						Please reconnect your Gmail account
					</p>
				{/if}
			</div>
		{/if}

		<!-- Actions -->
		<div class="flex gap-2 mt-auto">
			{#if !source.is_active}
				<button
					onclick={() => emailSourcesStore.connectGmail()}
					class="flex-1 px-3 py-2 text-sm font-medium bg-blue-600 text-white rounded hover:bg-blue-700"
				>
					Reconnect
				</button>
			{/if}
			<button
				onclick={handleDisconnect}
				disabled={isDisconnecting}
				class="flex-1 px-3 py-2 text-sm font-medium text-red-600 border border-red-300 rounded hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed"
			>
				{isDisconnecting ? 'Disconnecting...' : 'Disconnect'}
			</button>
		</div>
	</div>
</div>
