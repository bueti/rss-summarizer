// Core domain types matching backend models

export interface Feed {
	id: string;
	user_id: string;
	url: string;
	title: string;
	description: string;
	poll_frequency_minutes: number;
	last_polled_at?: string;
	is_active: boolean;
	created_at: string;
	updated_at: string;
}

export interface Article {
	id: string;
	feed_id: string;
	user_id: string;
	title: string;
	url: string;
	published_at?: string;
	original_content?: string;
	full_text?: string;
	summary?: string;
	key_points?: string[];
	importance_score?: number;
	topics?: string[];
	is_read: boolean;
	is_saved: boolean;
	is_archived: boolean;
	created_at: string;
	updated_at: string;
}

// Request types
export interface CreateFeedRequest {
	url: string;
	poll_frequency_minutes?: number;
}

export interface ArticleFilters {
	feed_id?: string;
	email_source_id?: string;
	min_importance?: number;
	topic?: string;
	is_read?: boolean;
	is_saved?: boolean;
	is_archived?: boolean;
	sort_by?: 'date' | 'importance';
	limit?: number;
	offset?: number;
}

export interface MarkAsReadRequest {
	is_read: boolean;
}

// Response wrapper types
export interface ListFeedsResponse {
	feeds: Feed[];
}

export interface ListArticlesResponse {
	articles: Article[];
}
