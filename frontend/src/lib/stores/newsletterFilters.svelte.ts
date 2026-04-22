import {
	listNewsletterFilters,
	createNewsletterFilter as apiCreateNewsletterFilter,
	updateNewsletterFilter as apiUpdateNewsletterFilter,
	deleteNewsletterFilter as apiDeleteNewsletterFilter
} from '$lib/api/generated';
import type { NewsletterFilter } from '$lib/api/generated';
import { browser } from '$app/environment';

class NewsletterFiltersStore {
	filters = $state<NewsletterFilter[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);

	async fetchFilters() {
		if (!browser) return;

		this.isLoading = true;
		this.error = null;

		try {
			const response = await listNewsletterFilters();
			if (response.status === 200) {
				this.filters = response.data.filters || [];
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to fetch newsletter filters';
			console.error('Error fetching newsletter filters:', err);
		} finally {
			this.isLoading = false;
		}
	}

	async createFilter(data: {
		email_source_id: string;
		name: string;
		sender_pattern: string;
		subject_pattern?: string;
	}): Promise<boolean> {
		if (!browser) return false;

		this.isLoading = true;
		this.error = null;

		try {
			const response = await apiCreateNewsletterFilter(data);
			if (response.status === 200) {
				this.filters = [...this.filters, response.data];
				return true;
			}
			return false;
		} catch (err: any) {
			this.error = err.message || 'Failed to create newsletter filter';
			console.error('Error creating newsletter filter:', err);
			throw err;
		} finally {
			this.isLoading = false;
		}
	}

	async updateFilter(
		id: string,
		updates: {
			name?: string;
			sender_pattern?: string;
			subject_pattern?: string;
			is_active?: boolean;
		}
	): Promise<boolean> {
		if (!browser) return false;

		this.isLoading = true;
		this.error = null;

		try {
			const response = await apiUpdateNewsletterFilter(id, updates);
			if (response.status === 200) {
				this.filters = this.filters.map((f) => (f.id === id ? response.data : f));
				return true;
			}
			return false;
		} catch (err: any) {
			this.error = err.message || 'Failed to update newsletter filter';
			console.error('Error updating newsletter filter:', err);
			throw err;
		} finally {
			this.isLoading = false;
		}
	}

	async deleteFilter(id: string): Promise<boolean> {
		if (!browser) return false;

		this.error = null;

		try {
			await apiDeleteNewsletterFilter(id);
			this.filters = this.filters.filter((f) => f.id !== id);
			return true;
		} catch (err: any) {
			this.error = err.message || 'Failed to delete newsletter filter';
			console.error('Error deleting newsletter filter:', err);
			throw err;
		}
	}

	getFiltersBySourceId(sourceId: string): NewsletterFilter[] {
		return this.filters.filter((f) => f.email_source_id === sourceId);
	}

	getFilterById(id: string): NewsletterFilter | undefined {
		return this.filters.find((f) => f.id === id);
	}

	get activeFiltersCount(): number {
		return this.filters.filter((f) => f.is_active).length;
	}
}

export const newsletterFiltersStore = new NewsletterFiltersStore();
