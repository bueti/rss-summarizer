<script lang="ts">
	let { status }: { status: string } = $props();

	const statusConfig: Record<
		string,
		{ label: string; class: string; icon: string }
	> = {
		pending: {
			label: 'Pending',
			class: 'bg-gray-100 text-gray-700 border-gray-200',
			icon: '○'
		},
		processing: {
			label: 'Processing',
			class: 'bg-blue-100 text-blue-700 border-blue-200',
			icon: '↻'
		},
		completed: {
			label: 'Completed',
			class: 'bg-green-100 text-green-800 border-green-200',
			icon: '✓'
		},
		failed: {
			label: 'Failed',
			class: 'bg-red-100 text-red-800 border-red-200',
			icon: '✕'
		}
	};

	const config = $derived(statusConfig[status] || statusConfig.pending);
</script>

<span
	class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium border {config.class}"
>
	<span class:animate-spin={status === 'processing'}>{config.icon}</span>
	<span>{config.label}</span>
</span>
