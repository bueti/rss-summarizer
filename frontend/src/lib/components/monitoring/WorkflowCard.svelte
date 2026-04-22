<script lang="ts">
	import type { WorkflowInfo } from '$lib/api/generated';

	let {
		workflow,
		status
	}: {
		workflow: WorkflowInfo;
		status: 'running' | 'success' | 'failed';
	} = $props();

	// Format duration
	function formatDuration(ms: number | null | undefined): string {
		if (!ms) return 'N/A';
		if (ms < 1000) return `${ms}ms`;
		if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
		return `${(ms / 60000).toFixed(1)}m`;
	}

	// Format time
	function formatTime(date: string | null | undefined): string {
		if (!date) return 'N/A';
		return new Date(date).toLocaleString();
	}

	// Get status styling
	const statusConfig = {
		running: {
			border: 'border-blue-500',
			badge: 'bg-blue-100 text-blue-800',
			icon: '↻',
			iconAnimate: 'animate-spin'
		},
		success: {
			border: 'border-green-500',
			badge: 'bg-green-100 text-green-800',
			icon: '✓',
			iconAnimate: ''
		},
		failed: {
			border: 'border-red-500',
			badge: 'bg-red-100 text-red-800',
			icon: '✕',
			iconAnimate: ''
		}
	};

	const config = $derived(statusConfig[status]);
</script>

<div
	class="bg-white rounded-lg shadow-sm border-l-4 {config.border} p-4 hover:shadow-md transition-shadow"
>
	<div class="flex items-start justify-between gap-4">
		<div class="flex-1 min-w-0">
			<!-- Workflow Type and Status -->
			<div class="flex items-center gap-2 mb-2">
				<span class="inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-medium {config.badge}">
					<span class={config.iconAnimate}>{config.icon}</span>
					{workflow.workflow_type}
				</span>
				<span class="text-xs text-gray-500 font-mono truncate">
					{workflow.workflow_id}
				</span>
			</div>

			<!-- Article Details (if available) -->
			{#if workflow.article_title}
				<div class="mb-2">
					<a
						href="/articles/{workflow.article_id}"
						class="text-base font-semibold text-gray-900 hover:text-blue-600 line-clamp-1"
					>
						{workflow.article_title}
					</a>
					{#if workflow.source_name}
						<p class="text-sm text-gray-600 mt-1">
							<span class="inline-flex items-center gap-1">
								{#if workflow.source_type === 'email'}
									<span class="text-purple-600">✉</span>
								{:else}
									<span class="text-blue-600">RSS</span>
								{/if}
								{workflow.source_name}
							</span>
						</p>
					{/if}
				</div>
			{:else if workflow.source_name}
				<div class="mb-2">
					<p class="text-base font-semibold text-gray-900">
						{workflow.workflow_type === 'ProcessFeedWorkflow' ? 'Feed' : 'Email Source'}: {workflow.source_name}
					</p>
				</div>
			{:else}
				<p class="text-sm text-gray-600 mb-2">System workflow</p>
			{/if}

			<!-- Timing Info -->
			<div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500">
				<span>Started: {formatTime(workflow.start_time)}</span>
				{#if workflow.close_time}
					<span>Completed: {formatTime(workflow.close_time)}</span>
				{/if}
				{#if workflow.execution_time_ms}
					<span class="font-medium">
						Duration: {formatDuration(workflow.execution_time_ms)}
					</span>
				{/if}
			</div>
		</div>

		<!-- Actions (if article exists) -->
		{#if workflow.article_id}
			<a
				href="/articles/{workflow.article_id}"
				class="px-3 py-1.5 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50 whitespace-nowrap"
			>
				View Article
			</a>
		{/if}
	</div>
</div>

<style>
	.line-clamp-1 {
		display: -webkit-box;
		-webkit-line-clamp: 1;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
