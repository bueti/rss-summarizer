import {
	listFeeds,
	createFeed as apiCreateFeed,
	updateFeed as apiUpdateFeed,
	deleteFeed as apiDeleteFeed,
	refreshFeed as apiRefreshFeed,
	getFeedHealth as apiGetFeedHealth
} from '$lib/api/generated';
import type { FeedResponseBody } from '$lib/api/generated';

type Feed = FeedResponseBody;

class FeedStore {
	feeds = $state<Feed[]>([]);
	totalCount = $state(0);
	currentPage = $state(1);
	pageSize = $state(20);
	isLoading = $state(false);
	error = $state<string | null>(null);

	async fetchFeeds(limit = 50, offset = 0) {
		this.isLoading = true;
		this.error = null;

		try {
			const response = await listFeeds({ limit, offset });
			if (response.status === 200) {
				this.feeds = response.data.feeds || [];
				this.totalCount = response.data.total_count || 0;
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to fetch feeds';
			console.error('Error fetching feeds:', err);
		} finally {
			this.isLoading = false;
		}
	}

	async createFeed(url: string, poll_frequency_minutes = 60) {
		this.isLoading = true;
		this.error = null;

		try {
			const response = await apiCreateFeed({
				url,
				poll_frequency_minutes
			});
			if (response.status === 200) {
				this.feeds = [...this.feeds, response.data];
				return response.data;
			}
			throw new Error('Failed to create feed');
		} catch (err: any) {
			this.error = err.message || 'Failed to create feed';
			console.error('Error creating feed:', err);
			throw err;
		} finally {
			this.isLoading = false;
		}
	}

	async updateFeed(
		id: string,
		updates: {
			title?: string;
			poll_frequency_minutes?: number;
			is_active?: boolean;
		}
	) {
		this.isLoading = true;
		this.error = null;

		try {
			const response = await apiUpdateFeed(id, updates);
			if (response.status === 200) {
				this.feeds = this.feeds.map((f) => (f.id === id ? response.data : f));
				return response.data;
			}
			throw new Error('Failed to update feed');
		} catch (err: any) {
			this.error = err.message || 'Failed to update feed';
			console.error('Error updating feed:', err);
			throw err;
		} finally {
			this.isLoading = false;
		}
	}

	async refreshFeed(id: string) {
		try {
			await apiRefreshFeed(id);
		} catch (err: any) {
			this.error = err.message || 'Failed to refresh feed';
			console.error('Error refreshing feed:', err);
			throw err;
		}
	}

	async getFeedHealth(id: string) {
		try {
			const response = await apiGetFeedHealth(id);
			if (response.status === 200) {
				return response.data;
			}
			return null;
		} catch (err: any) {
			this.error = err.message || 'Failed to get feed health';
			console.error('Error getting feed health:', err);
			return null;
		}
	}

	async deleteFeed(id: string) {
		this.isLoading = true;
		this.error = null;

		try {
			await apiDeleteFeed(id);
			this.feeds = this.feeds.filter((f) => f.id !== id);
		} catch (err: any) {
			this.error = err.message || 'Failed to delete feed';
			console.error('Error deleting feed:', err);
			throw err;
		} finally {
			this.isLoading = false;
		}
	}

	getFeedById(id: string): Feed | undefined {
		return this.feeds.find((f) => f.id === id);
	}

	async fetchPage(page: number) {
		this.currentPage = page;
		const offset = (page - 1) * this.pageSize;
		await this.fetchFeeds(this.pageSize, offset);
	}

	get totalPages(): number {
		return Math.ceil(this.totalCount / this.pageSize);
	}
}

export const feedStore = new FeedStore();
