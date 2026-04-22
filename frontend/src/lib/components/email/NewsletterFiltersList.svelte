<script lang="ts">
	import type { NewsletterFilter } from '$lib/api/generated';
	import { newsletterFiltersStore } from '$lib/stores/newsletterFilters.svelte';

	let { sourceId }: { sourceId?: string } = $props();

	// Filter to show only filters for the specified source, or all if no source specified
	const filters = $derived(
		sourceId
			? newsletterFiltersStore.getFiltersBySourceId(sourceId)
			: newsletterFiltersStore.filters
	);

	async function handleToggleActive(filterId: string, currentActive: boolean) {
		try {
			await newsletterFiltersStore.updateFilter(filterId, {
				is_active: !currentActive
			});
		} catch (err) {
			console.error('Failed to toggle filter:', err);
			alert('Failed to toggle filter');
		}
	}

	async function handleDelete(filterId: string, filterName: string) {
		if (!confirm(`Delete filter "${filterName}"? This cannot be undone.`)) {
			return;
		}

		try {
			await newsletterFiltersStore.deleteFilter(filterId);
		} catch (err) {
			console.error('Failed to delete filter:', err);
			alert('Failed to delete filter');
		}
	}

	function getStatusColor(isActive: boolean) {
		return isActive ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800';
	}
</script>

<div class="space-y-3">
	{#if filters.length === 0}
		<div class="text-center py-8 bg-gray-50 rounded-lg border border-gray-200">
			<p class="text-gray-500">No newsletter filters configured</p>
			<p class="text-sm text-gray-400 mt-1">
				Add a filter to start receiving and analyzing newsletters
			</p>
		</div>
	{:else}
		{#each filters as filter}
			<div
				class="bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow"
			>
				<div class="flex items-start justify-between">
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2 mb-2">
							<h4 class="text-base font-semibold text-gray-900">{filter.name}</h4>
							<span
								class="px-2 py-0.5 text-xs font-medium rounded-full {getStatusColor(
									filter.is_active
								)}"
							>
								{filter.is_active ? 'Active' : 'Inactive'}
							</span>
						</div>

						<div class="space-y-1 text-sm text-gray-600">
							<div class="flex items-center gap-2">
								<span class="font-medium text-gray-700">Sender:</span>
								<code class="bg-gray-100 px-2 py-0.5 rounded text-xs"
									>{filter.sender_pattern}</code
								>
							</div>
							{#if filter.subject_pattern}
								<div class="flex items-center gap-2">
									<span class="font-medium text-gray-700">Subject:</span>
									<code class="bg-gray-100 px-2 py-0.5 rounded text-xs"
										>{filter.subject_pattern}</code
									>
								</div>
							{/if}
						</div>
					</div>

					<!-- Actions -->
					<div class="flex gap-2 ml-4">
						<button
							onclick={() => handleToggleActive(filter.id, filter.is_active)}
							class="px-3 py-1 text-xs font-medium border border-gray-300 rounded hover:bg-gray-50"
							title={filter.is_active ? 'Deactivate filter' : 'Activate filter'}
						>
							{filter.is_active ? 'Deactivate' : 'Activate'}
						</button>
						<button
							onclick={() => handleDelete(filter.id, filter.name)}
							class="px-3 py-1 text-xs font-medium text-red-600 border border-red-300 rounded hover:bg-red-50"
							title="Delete filter"
						>
							Delete
						</button>
					</div>
				</div>
			</div>
		{/each}
	{/if}
</div>
