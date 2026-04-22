import {
	listTopics,
	createTopic,
	updateTopicPreference,
	deleteTopic
} from '$lib/api/generated';
import type { TopicWithPreference } from '$lib/api/generated';
import { browser } from '$app/environment';

type TopicPreference = 'high' | 'normal' | 'hide';

class TopicStore {
	topics = $state<TopicWithPreference[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);

	async fetch() {
		if (!browser) return;

		this.isLoading = true;
		this.error = null;

		try {
			const response = await listTopics();
			if (response.status === 200) {
				this.topics = response.data.topics || [];
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to fetch topics';
			console.error('Error fetching topics:', err);
		} finally {
			this.isLoading = false;
		}
	}

	// Alias for compatibility
	async fetchTopics() {
		return this.fetch();
	}

	async create(name: string, preference: TopicPreference = 'normal'): Promise<boolean> {
		if (!browser) return false;

		this.isLoading = true;
		this.error = null;

		try {
			await createTopic({ name });

			// Refresh topics to get the created topic with preference
			await this.fetch();

			return true;
		} catch (err: any) {
			this.error = err.message || 'Failed to create topic';
			console.error('Error creating topic:', err);
			return false;
		} finally {
			this.isLoading = false;
		}
	}

	async updatePreference(id: string, preference: TopicPreference): Promise<boolean> {
		if (!browser) return false;

		this.error = null;

		try {
			await updateTopicPreference(id, { preference });

			// Update local state
			const index = this.topics.findIndex((t) => t.id === id);
			if (index !== -1) {
				this.topics[index] = { ...this.topics[index], preference };
			}

			return true;
		} catch (err: any) {
			this.error = err.message || 'Failed to update topic preference';
			console.error('Error updating topic preference:', err);
			return false;
		}
	}

	async delete(id: string): Promise<boolean> {
		if (!browser) return false;

		this.error = null;

		try {
			await deleteTopic(id);

			// Remove from local state
			this.topics = this.topics.filter((t) => t.id !== id);

			return true;
		} catch (err: any) {
			this.error = err.message || 'Failed to delete topic';
			console.error('Error deleting topic:', err);
			return false;
		}
	}

	// Get unique topic names sorted alphabetically
	get topicNames(): string[] {
		return this.topics.map((t) => t.name).sort();
	}
}

export const topicsStore = new TopicStore();
