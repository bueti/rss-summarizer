import { getUserPreferences, updateUserPreferences } from '$lib/api/generated';
import type { PreferencesResponse, UpdatePreferencesRequestBody } from '$lib/api/generated';

type UserPreferences = PreferencesResponse;

class PreferencesStore {
	preferences = $state<UserPreferences | null>(null);
	isLoading = $state(false);
	error = $state<string | null>(null);

	async fetch() {
		this.isLoading = true;
		this.error = null;

		try {
			const response = await getUserPreferences();
			if (response.status === 200) {
				this.preferences = response.data;
			}
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to load preferences';
			console.error('Failed to fetch preferences:', err);
		} finally {
			this.isLoading = false;
		}
	}

	async update(data: UpdatePreferencesRequestBody) {
		this.isLoading = true;
		this.error = null;

		try {
			const response = await updateUserPreferences(data);
			if (response.status === 200) {
				this.preferences = response.data;
				return true;
			}
			return false;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to update preferences';
			console.error('Failed to update preferences:', err);
			return false;
		} finally {
			this.isLoading = false;
		}
	}
}

export const preferencesStore = new PreferencesStore();
