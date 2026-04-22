import {
	listEmailSources,
	deleteEmailSource as apiDeleteEmailSource
} from '$lib/api/generated';
import type { EmailSource } from '$lib/api/generated';
import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';

const API_BASE = env.PUBLIC_API_URL || import.meta.env.VITE_API_URL || 'http://localhost:8080';

class EmailSourcesStore {
	sources = $state<EmailSource[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);

	async fetchSources() {
		if (!browser) return;

		this.isLoading = true;
		this.error = null;

		try {
			const response = await listEmailSources();
			if (response.status === 200) {
				this.sources = response.data.email_sources || [];
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to fetch email sources';
			console.error('Error fetching email sources:', err);
		} finally {
			this.isLoading = false;
		}
	}

	async connectGmail() {
		if (!browser) return;

		this.error = null;

		try {
			// First, fetch the auth URL from the backend
			const response = await fetch(`${API_BASE}/v1/auth/gmail/connect`, {
				credentials: 'include' // Include auth cookie
			});

			if (!response.ok) {
				throw new Error('Failed to initiate Gmail connection');
			}

			const data = await response.json();

			if (!data.auth_url) {
				throw new Error('No auth URL returned from server');
			}

			// Open the Google OAuth URL in a popup
			const authWindow = window.open(
				data.auth_url,
				'gmail-oauth',
				'width=600,height=700,scrollbars=yes'
			);

			if (!authWindow) {
				this.error = 'Failed to open OAuth window. Please allow popups.';
				return;
			}

			// Poll for successful connection (window will close on success)
			const pollInterval = setInterval(() => {
				if (authWindow.closed) {
					clearInterval(pollInterval);
					// Refresh sources after connection
					this.fetchSources();
				}
			}, 500);
		} catch (err: any) {
			this.error = err.message || 'Failed to initiate Gmail connection';
			console.error('Error connecting Gmail:', err);
		}
	}

	async disconnect(id: string) {
		if (!browser) return;

		this.error = null;

		try {
			await apiDeleteEmailSource(id);
			this.sources = this.sources.filter((s) => s.id !== id);
		} catch (err: any) {
			this.error = err.message || 'Failed to disconnect email source';
			console.error('Error disconnecting email source:', err);
			throw err;
		}
	}

	getSourceById(id: string): EmailSource | undefined {
		return this.sources.find((s) => s.id === id);
	}

	get hasActiveSource(): boolean {
		return this.sources.some((s) => s.is_active);
	}
}

export const emailSourcesStore = new EmailSourcesStore();
