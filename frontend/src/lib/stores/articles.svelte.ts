import {
	listArticles,
	getArticle,
	markArticleRead,
	setArticleSaved,
	setArticleArchived,
	bulkMarkRead,
	bulkSetSaved,
	bulkSetArchived,
	processArticle
} from '$lib/api/generated';
import type { ArticleResponse, ListArticlesParams } from '$lib/api/generated';

type Article = ArticleResponse;

interface ArticleFilters {
	feed_id?: string;
	email_source_id?: string;
	min_importance?: number;
	topic?: string;
	is_read?: boolean;
	is_saved?: boolean;
	is_archived?: boolean;
	processing_status?: string;
	sort_by?: 'date' | 'importance';
	limit?: number;
	offset?: number;
}

class ArticleStore {
	articles = $state<Article[]>([]);
	totalCount = $state(0);
	currentPage = $state(1);
	pageSize = $state(20);
	isLoading = $state(false);
	error = $state<string | null>(null);
	currentFilters = $state<ArticleFilters>({});

	async fetchArticles(filters?: ArticleFilters) {
		this.isLoading = true;
		this.error = null;
		this.currentFilters = filters || {};

		try {
			const params: ListArticlesParams = {
				feed_id: filters?.feed_id,
				email_source_id: filters?.email_source_id,
				min_importance: filters?.min_importance,
				topic: filters?.topic,
				is_read: filters?.is_read !== undefined ? String(filters.is_read) : undefined,
				is_saved: filters?.is_saved !== undefined ? String(filters.is_saved) : undefined,
				is_archived: filters?.is_archived !== undefined ? String(filters.is_archived) : undefined,
				processing_status: filters?.processing_status,
				sort_by: filters?.sort_by,
				limit: filters?.limit || 50,
				offset: filters?.offset || 0
			};

			const response = await listArticles(params);
			if (response.status === 200) {
				this.articles = response.data.articles || [];
				this.totalCount = response.data.total_count || 0;
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to fetch articles';
			console.error('Error fetching articles:', err);
		} finally {
			this.isLoading = false;
		}
	}

	async markAsRead(id: string, isRead: boolean = true) {
		try {
			await markArticleRead(id, { is_read: isRead });

			// If filtering for unread articles and marking as read, remove from list
			if (this.currentFilters.is_read === false && isRead) {
				this.articles = this.articles.filter((a) => a.id !== id);
				this.totalCount = Math.max(0, this.totalCount - 1);
			}
			// If filtering for read articles and marking as unread, remove from list
			else if (this.currentFilters.is_read === true && !isRead) {
				this.articles = this.articles.filter((a) => a.id !== id);
				this.totalCount = Math.max(0, this.totalCount - 1);
			}
			// Otherwise just update the article state
			else {
				const article = this.articles.find((a) => a.id === id);
				if (article) {
					article.is_read = isRead;
				}
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to mark article as read';
			console.error('Error marking article as read:', err);
			throw err;
		}
	}

	async bulkMarkAsRead(articleIds: string[], isRead: boolean = true) {
		try {
			await bulkMarkRead({
				article_ids: articleIds,
				is_read: isRead
			});

			// If filtering for unread articles and marking as read, remove from list
			if (this.currentFilters.is_read === false && isRead) {
				this.articles = this.articles.filter((a) => !articleIds.includes(a.id));
				this.totalCount = Math.max(0, this.totalCount - articleIds.length);
			}
			// If filtering for read articles and marking as unread, remove from list
			else if (this.currentFilters.is_read === true && !isRead) {
				this.articles = this.articles.filter((a) => !articleIds.includes(a.id));
				this.totalCount = Math.max(0, this.totalCount - articleIds.length);
			}
			// Otherwise just update the article states
			else {
				this.articles = this.articles.map((a) =>
					articleIds.includes(a.id) ? { ...a, is_read: isRead } : a
				);
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to bulk mark articles';
			console.error('Error bulk marking articles:', err);
			throw err;
		}
	}

	async retryArticle(id: string) {
		try {
			await processArticle(id);

			// Update local state to show it's pending
			const article = this.articles.find((a) => a.id === id);
			if (article) {
				article.processing_status = 'pending';
				article.processing_error = undefined;
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to retry article';
			console.error('Error retrying article:', err);
			throw err;
		}
	}

	async getArticleById(id: string): Promise<Article | null> {
		try {
			const response = await getArticle(id);
			if (response.status === 200) {
				return response.data;
			}
			return null;
		} catch (err: any) {
			this.error = err.message || 'Failed to fetch article';
			console.error('Error fetching article:', err);
			return null;
		}
	}

	async fetchPage(page: number) {
		this.currentPage = page;
		const offset = (page - 1) * this.pageSize;
		await this.fetchArticles({
			...this.currentFilters,
			limit: this.pageSize,
			offset
		});
	}

	// Derived getters
	get unreadCount(): number {
		return this.articles.filter((a) => !a.is_read).length;
	}

	get allTopics(): string[] {
		const topicsSet = new Set<string>();
		this.articles.forEach((a) => {
			a.topics?.forEach((topic) => topicsSet.add(topic));
		});
		return Array.from(topicsSet).sort();
	}

	get totalPages(): number {
		return Math.ceil(this.totalCount / this.pageSize);
	}

	async fetchUnreadArticles(): Promise<Article[]> {
		try {
			const response = await listArticles({
				is_read: 'false',
				limit: 100
			});
			if (response.status === 200) {
				return response.data.articles || [];
			}
			return [];
		} catch (err: any) {
			console.error('Error fetching unread articles:', err);
			return [];
		}
	}

	async setSaved(id: string, isSaved: boolean = true) {
		try {
			await setArticleSaved(id, { is_saved: isSaved });

			// If filtering for saved articles and unsaving, remove from list
			if (this.currentFilters.is_saved === true && !isSaved) {
				this.articles = this.articles.filter((a) => a.id !== id);
				this.totalCount = Math.max(0, this.totalCount - 1);
			}
			// Otherwise just update the article state
			else {
				const article = this.articles.find((a) => a.id === id);
				if (article) {
					article.is_saved = isSaved;
				}
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to set saved status';
			console.error('Error setting saved status:', err);
			throw err;
		}
	}

	async setArchived(id: string, isArchived: boolean = true) {
		try {
			await setArticleArchived(id, { is_archived: isArchived });

			// If archiving, remove from current view (unless viewing archive)
			if (isArchived && this.currentFilters.is_archived !== true) {
				this.articles = this.articles.filter((a) => a.id !== id);
				this.totalCount = Math.max(0, this.totalCount - 1);
			}
			// If unarchiving from archive view, remove from list
			else if (!isArchived && this.currentFilters.is_archived === true) {
				this.articles = this.articles.filter((a) => a.id !== id);
				this.totalCount = Math.max(0, this.totalCount - 1);
			}
			// Otherwise just update the article state
			else {
				const article = this.articles.find((a) => a.id === id);
				if (article) {
					article.is_archived = isArchived;
				}
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to set archived status';
			console.error('Error setting archived status:', err);
			throw err;
		}
	}

	async bulkSetSaved(articleIds: string[], isSaved: boolean = true) {
		try {
			await bulkSetSaved({
				article_ids: articleIds,
				is_saved: isSaved
			});

			// If filtering for saved articles and unsaving, remove from list
			if (this.currentFilters.is_saved === true && !isSaved) {
				this.articles = this.articles.filter((a) => !articleIds.includes(a.id));
				this.totalCount = Math.max(0, this.totalCount - articleIds.length);
			}
			// Otherwise just update the article states
			else {
				this.articles = this.articles.map((a) =>
					articleIds.includes(a.id) ? { ...a, is_saved: isSaved } : a
				);
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to bulk set saved status';
			console.error('Error bulk setting saved status:', err);
			throw err;
		}
	}

	async bulkSetArchived(articleIds: string[], isArchived: boolean = true) {
		try {
			await bulkSetArchived({
				article_ids: articleIds,
				is_archived: isArchived
			});

			// If archiving, remove from current view (unless viewing archive)
			if (isArchived && this.currentFilters.is_archived !== true) {
				this.articles = this.articles.filter((a) => !articleIds.includes(a.id));
				this.totalCount = Math.max(0, this.totalCount - articleIds.length);
			}
			// If unarchiving from archive view, remove from list
			else if (!isArchived && this.currentFilters.is_archived === true) {
				this.articles = this.articles.filter((a) => !articleIds.includes(a.id));
				this.totalCount = Math.max(0, this.totalCount - articleIds.length);
			}
			// Otherwise just update the article states
			else {
				this.articles = this.articles.map((a) =>
					articleIds.includes(a.id) ? { ...a, is_archived: isArchived } : a
				);
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to bulk set archived status';
			console.error('Error bulk setting archived status:', err);
			throw err;
		}
	}
}

export const articleStore = new ArticleStore();
