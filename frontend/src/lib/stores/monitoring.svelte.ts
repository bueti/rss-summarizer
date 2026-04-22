import { getWorkflowOverview } from '$lib/api/generated';
import type { WorkflowInfo } from '$lib/api/generated';

class MonitoringStore {
	running = $state<WorkflowInfo[]>([]);
	recentSuccess = $state<WorkflowInfo[]>([]);
	recentFailed = $state<WorkflowInfo[]>([]);

	totalRunning = $state(0);
	totalSuccess24h = $state(0);
	totalFailed24h = $state(0);

	isLoading = $state(false);
	error = $state<string | null>(null);
	lastUpdated = $state<Date | null>(null);

	async fetchWorkflows() {
		// Don't flash loading spinner on refresh
		const isFirstLoad = this.running.length === 0;
		if (isFirstLoad) {
			this.isLoading = true;
		}

		this.error = null;

		try {
			const response = await getWorkflowOverview();

			if (response.status === 200) {
				this.running = response.data.running || [];
				this.recentSuccess = response.data.recent_success || [];
				this.recentFailed = response.data.recent_failed || [];
				this.totalRunning = response.data.total_running || 0;
				this.totalSuccess24h = response.data.total_success_24h || 0;
				this.totalFailed24h = response.data.total_failed_24h || 0;
				this.lastUpdated = new Date();
			}
		} catch (err: any) {
			this.error = err.message || 'Failed to fetch workflow data';
			console.error('Error fetching workflows:', err);
		} finally {
			if (isFirstLoad) {
				this.isLoading = false;
			}
		}
	}
}

export const monitoringStore = new MonitoringStore();
