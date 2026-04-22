<script lang="ts">
	let { status }: { status: string } = $props();

	const statusConfig: Record<
		string,
		{ label: string; class: string; icon: string }
	> = {
		healthy: {
			label: 'Healthy',
			class: 'bg-green-100 text-green-800 border-green-200',
			icon: '✓'
		},
		warning: {
			label: 'Warning',
			class: 'bg-yellow-100 text-yellow-800 border-yellow-200',
			icon: '⚠'
		},
		error: {
			label: 'Error',
			class: 'bg-red-100 text-red-800 border-red-200',
			icon: '✕'
		},
		paused: {
			label: 'Paused',
			class: 'bg-gray-100 text-gray-800 border-gray-200',
			icon: '⏸'
		}
	};

	const config = $derived(statusConfig[status] || statusConfig.healthy);
</script>

<span
	class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium border {config.class}"
>
	<span>{config.icon}</span>
	<span>{config.label}</span>
</span>
