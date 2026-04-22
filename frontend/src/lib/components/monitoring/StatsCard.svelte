<script lang="ts">
	let {
		title,
		count,
		color,
		icon
	}: {
		title: string;
		count: number;
		color: 'blue' | 'green' | 'red';
		icon: string;
	} = $props();

	const colorClasses = {
		blue: 'bg-blue-50 border-blue-200 text-blue-600',
		green: 'bg-green-50 border-green-200 text-green-600',
		red: 'bg-red-50 border-red-200 text-red-600'
	};

	const bgClass = $derived(colorClasses[color]);
</script>

<div class="bg-white rounded-lg shadow-sm border-2 {bgClass} p-6">
	<div class="flex items-center justify-between">
		<div>
			<p class="text-sm font-medium text-gray-600">{title}</p>
			<p class="text-3xl font-bold mt-2">{count}</p>
		</div>
		<div class="text-4xl opacity-50">{icon}</div>
	</div>
</div>
