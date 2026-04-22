import { me, googleLogin, logout as apiLogout } from '$lib/api/generated';
import type { MeResponseBody } from '$lib/api/generated';

type User = MeResponseBody;

class AuthStore {
	user = $state<User | null>(null);
	isLoading = $state(true);
	isAuthenticated = $derived(this.user !== null);
	isAdmin = $derived(this.user?.role === 'admin');
	error = $state<string | null>(null);

	async initialize() {
		this.isLoading = true;
		this.error = null;

		try {
			console.log('[AuthStore] Calling /auth/me...');
			const response = await me();
			console.log('[AuthStore] Response:', response);
			if (response.status === 200) {
				console.log('[AuthStore] Setting user:', response.data);
				this.user = response.data;
			} else {
				console.log('[AuthStore] Non-200 status:', response.status);
			}
		} catch (err: any) {
			console.error('[AuthStore] Auth initialization error:', err);
			this.user = null;
		} finally {
			this.isLoading = false;
			console.log('[AuthStore] Initialize complete. isAuthenticated:', this.isAuthenticated);
		}
	}

	async login() {
		try {
			const response = await googleLogin();

			// Redirect to Google OAuth
			if (response.status === 200 && response.data.auth_url) {
				window.location.href = response.data.auth_url;
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to login';
			console.error('Login error:', err);
		}
	}

	async logout() {
		try {
			await apiLogout();
			this.user = null;
			window.location.href = '/login';
		} catch (err: any) {
			console.error('Logout error:', err);
			// Clear user anyway
			this.user = null;
			window.location.href = '/login';
		}
	}
}

export const authStore = new AuthStore();
