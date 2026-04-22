<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { monitoringStore } from '$lib/stores/monitoring.svelte';
	import WorkflowCard from '$lib/components/monitoring/WorkflowCard.svelte';
	import StatsCard from '$lib/components/monitoring/StatsCard.svelte';

	let refreshInterval: ReturnType<typeof setInterval> | undefined = $state(undefined);
	const REFRESH_INTERVAL = 5000; // 5 seconds

	onMount(async () => {
		// Initial fetch
		await monitoringStore.fetchWorkflows();

		// Setup auto-refresh
		refreshInterval = setInterval(() => {
			monitoringStore.fetchWorkflows();
		}, REFRESH_INTERVAL);
	});

	onDestroy(() => {
		if (refreshInterval !== undefined) {
			clearInterval(refreshInterval);
		}
	});

	// Derived computed values
	const runningCount = $derived(monitoringStore.running.length);
	const successCount = $derived(monitoringStore.totalSuccess24h);
	const failedCount = $derived(monitoringStore.totalFailed24h);
</script>

<div class="space-y-6">
	<!-- Header with stats -->
	<div>
		<h1 class="text-3xl font-bold text-gray-900">Workflow Monitoring</h1>
		<p class="text-gray-600 mt-1">Real-time Temporal workflow status</p>
	</div>

	<!-- Stats cards -->
	<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
		<StatsCard title="Running" count={runningCount} color="blue" icon="↻" />
		<StatsCard title="Completed (24h)" count={successCount} color="green" icon="✓" />
		<StatsCard title="Failed (24h)" count={failedCount} color="red" icon="✕" />
	</div>

	<!-- Error display -->
	{#if monitoringStore.error}
		<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
			{monitoringStore.error}
		</div>
	{/if}

	<!-- Loading state -->
	{#if monitoringStore.isLoading && monitoringStore.running.length === 0}
		<div class="text-center py-12">
			<div class="animate-spin text-4xl">↻</div>
			<p class="text-gray-600 mt-4">Loading workflows...</p>
		</div>
	{:else}
		<!-- Running workflows section -->
		<section>
			<h2 class="text-2xl font-semibold mb-4 flex items-center gap-2">
				<span class="text-blue-600">↻</span>
				Running Workflows
				{#if monitoringStore.isLoading}
					<span class="text-sm text-gray-500">(refreshing...)</span>
				{/if}
			</h2>
			{#if runningCount === 0}
				<div class="bg-gray-50 border border-gray-200 rounded-lg p-6 text-center">
					<p class="text-gray-600">No workflows currently running</p>
				</div>
			{:else}
				<div class="space-y-3">
					{#each monitoringStore.running as workflow}
						<WorkflowCard {workflow} status="running" />
					{/each}
				</div>
			{/if}
		</section>

		<!-- Recent successes -->
		<section>
			<h2 class="text-2xl font-semibold mb-4 flex items-center gap-2">
				<span class="text-green-600">✓</span>
				Recently Completed (24h)
			</h2>
			{#if monitoringStore.recentSuccess.length === 0}
				<div class="bg-gray-50 border border-gray-200 rounded-lg p-6 text-center">
					<p class="text-gray-600">No completed workflows in last 24 hours</p>
				</div>
			{:else}
				<div class="space-y-3">
					{#each monitoringStore.recentSuccess.slice(0, 20) as workflow}
						<WorkflowCard {workflow} status="success" />
					{/each}
				</div>
			{/if}
		</section>

		<!-- Recent failures -->
		<section>
			<h2 class="text-2xl font-semibold mb-4 flex items-center gap-2">
				<span class="text-red-600">✕</span>
				Recently Failed (24h)
			</h2>
			{#if monitoringStore.recentFailed.length === 0}
				<div class="bg-gray-50 border border-gray-200 rounded-lg p-6 text-center">
					<p class="text-gray-600">No failed workflows in last 24 hours</p>
				</div>
			{:else}
				<div class="space-y-3">
					{#each monitoringStore.recentFailed as workflow}
						<WorkflowCard {workflow} status="failed" />
					{/each}
				</div>
			{/if}
		</section>
	{/if}

	<!-- Last updated timestamp -->
	<div class="text-center text-sm text-gray-500">
		Last updated: {monitoringStore.lastUpdated?.toLocaleTimeString() || 'Never'}
		· Auto-refreshes every {REFRESH_INTERVAL / 1000}s
	</div>
</div>
