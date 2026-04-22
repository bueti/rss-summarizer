<script lang="ts">
	import type { EmailSource } from '$lib/api/generated';
	import { newsletterFiltersStore } from '$lib/stores/newsletterFilters.svelte';

	let {
		emailSources,
		onClose
	}: {
		emailSources: EmailSource[];
		onClose?: () => void;
	} = $props();

	let selectedSourceId = $state('');
	let name = $state('');
	let senderPattern = $state('');
	let subjectPattern = $state('');
	let isSubmitting = $state(false);
	let error = $state<string | null>(null);

	// Initialize selected source from emailSources prop
	$effect(() => {
		selectedSourceId = emailSources[0]?.id || '';
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = null;

		if (!selectedSourceId) {
			error = 'Please select an email source';
			return;
		}

		if (!name.trim()) {
			error = 'Please enter a filter name';
			return;
		}

		if (!senderPattern.trim()) {
			error = 'Please enter a sender pattern';
			return;
		}

		isSubmitting = true;

		try {
			await newsletterFiltersStore.createFilter({
				email_source_id: selectedSourceId,
				name: name.trim(),
				sender_pattern: senderPattern.trim(),
				subject_pattern: subjectPattern.trim() || undefined
			});

			// Reset form
			name = '';
			senderPattern = '';
			subjectPattern = '';

			onClose?.();
		} catch (err: any) {
			error = err.message || 'Failed to create newsletter filter';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="bg-white rounded-lg p-6 max-w-md w-full">
	<h2 class="text-xl font-semibold text-gray-900 mb-4">Add Newsletter Filter</h2>

	<form onsubmit={handleSubmit} class="space-y-4">
		<!-- Email Source Selection -->
		<div>
			<label for="email-source" class="block text-sm font-medium text-gray-700 mb-1">
				Email Account
			</label>
			<select
				id="email-source"
				bind:value={selectedSourceId}
				required
				class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
			>
				{#each emailSources as source}
					<option value={source.id}>{source.email_address}</option>
				{/each}
			</select>
		</div>

		<!-- Filter Name -->
		<div>
			<label for="filter-name" class="block text-sm font-medium text-gray-700 mb-1">
				Filter Name
			</label>
			<input
				id="filter-name"
				type="text"
				bind:value={name}
				placeholder="e.g., Substack Newsletters"
				required
				class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
			/>
		</div>

		<!-- Sender Pattern -->
		<div>
			<label for="sender-pattern" class="block text-sm font-medium text-gray-700 mb-1">
				Sender Pattern
			</label>
			<input
				id="sender-pattern"
				type="text"
				bind:value={senderPattern}
				placeholder="e.g., *@substack.com"
				required
				class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
			/>
			<p class="text-xs text-gray-500 mt-1">
				Examples: *@substack.com (domain), newsletter@example.com (exact)
			</p>
		</div>

		<!-- Subject Pattern (Optional) -->
		<div>
			<label for="subject-pattern" class="block text-sm font-medium text-gray-700 mb-1">
				Subject Pattern (Optional)
			</label>
			<input
				id="subject-pattern"
				type="text"
				bind:value={subjectPattern}
				placeholder="e.g., ^Weekly Digest"
				class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
			/>
			<p class="text-xs text-gray-500 mt-1">Supports regex (leave empty to match all subjects)</p>
		</div>

		<!-- Error Message -->
		{#if error}
			<div class="bg-red-50 border border-red-200 rounded p-3">
				<p class="text-sm text-red-800">{error}</p>
			</div>
		{/if}

		<!-- Actions -->
		<div class="flex gap-2 pt-2">
			<button
				type="submit"
				disabled={isSubmitting}
				class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed font-medium"
			>
				{isSubmitting ? 'Adding...' : 'Add Filter'}
			</button>
			{#if onClose}
				<button
					type="button"
					onclick={onClose}
					class="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 font-medium"
				>
					Cancel
				</button>
			{/if}
		</div>
	</form>
</div>
