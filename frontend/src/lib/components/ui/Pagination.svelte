<script lang="ts">
	let {
		currentPage,
		totalPages,
		onPageChange
	}: {
		currentPage: number;
		totalPages: number;
		onPageChange: (page: number) => void;
	} = $props();

	function goToPage(page: number) {
		if (page >= 1 && page <= totalPages) {
			onPageChange(page);
		}
	}

	const visiblePages = $derived.by(() => {
		const pages: (number | string)[] = [];
		const maxVisible = 7;

		if (totalPages <= maxVisible) {
			// Show all pages if total is small
			for (let i = 1; i <= totalPages; i++) {
				pages.push(i);
			}
		} else {
			// Always show first page
			pages.push(1);

			if (currentPage > 3) {
				pages.push('...');
			}

			// Show pages around current
			const start = Math.max(2, currentPage - 1);
			const end = Math.min(totalPages - 1, currentPage + 1);

			for (let i = start; i <= end; i++) {
				pages.push(i);
			}

			if (currentPage < totalPages - 2) {
				pages.push('...');
			}

			// Always show last page
			if (totalPages > 1) {
				pages.push(totalPages);
			}
		}

		return pages;
	});
</script>

{#if totalPages > 1}
	<nav class="flex items-center justify-center gap-2 py-4" aria-label="Pagination">
		<!-- Previous Button -->
		<button
			onclick={() => goToPage(currentPage - 1)}
			disabled={currentPage === 1}
			class="px-3 py-2 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
			aria-label="Previous page"
		>
			← Previous
		</button>

		<!-- Page Numbers -->
		<div class="flex gap-1">
			{#each visiblePages as page}
				{#if page === '...'}
					<span class="px-3 py-2 text-gray-500">...</span>
				{:else}
					<button
						onclick={() => goToPage(page as number)}
						class="px-3 py-2 text-sm font-medium border rounded {currentPage === page
							? 'bg-blue-600 text-white border-blue-600'
							: 'border-gray-300 hover:bg-gray-50'}"
						aria-label={`Page ${page}`}
						aria-current={currentPage === page ? 'page' : undefined}
					>
						{page}
					</button>
				{/if}
			{/each}
		</div>

		<!-- Next Button -->
		<button
			onclick={() => goToPage(currentPage + 1)}
			disabled={currentPage === totalPages}
			class="px-3 py-2 text-sm font-medium border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
			aria-label="Next page"
		>
			Next →
		</button>
	</nav>
{/if}
