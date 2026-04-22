<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { adminListUsers, adminUpdateUserRole } from '$lib/api/generated';
	import type { UserResponse } from '$lib/api/generated';

	type User = UserResponse;

	let users = $state<User[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let updatingUserId = $state<string | null>(null);

	onMount(async () => {
		// Check if user is admin
		if (!authStore.isAdmin) {
			goto('/');
			return;
		}

		await fetchUsers();
	});

	async function fetchUsers() {
		isLoading = true;
		error = null;

		try {
			const response = await adminListUsers();
			if (response.status === 200) {
				users = response.data.users || [];
			}
		} catch (err: any) {
			error = err.message || 'Failed to load users';
			console.error('Error fetching users:', err);
		} finally {
			isLoading = false;
		}
	}

	async function updateUserRole(userId: string, newRole: string) {
		updatingUserId = userId;
		error = null;

		try {
			const response = await adminUpdateUserRole(userId, {
				role: newRole as 'user' | 'admin'
			});

			// Update local state
			if (response.status === 200) {
				const index = users.findIndex((u) => u.id === userId);
				if (index !== -1) {
					users[index] = response.data;
				}
			}
		} catch (err: any) {
			error = err.message || 'Failed to update user role';
			console.error('Error updating user role:', err);
		} finally {
			updatingUserId = null;
		}
	}
</script>

<svelte:head>
	<title>User Management - RSS Summarizer</title>
</svelte:head>

<div class="max-w-6xl mx-auto px-4 py-8">
	<h1 class="text-3xl font-bold text-gray-900 mb-8">User Management</h1>

	{#if error}
		<div class="rounded-md bg-red-50 p-4 mb-6">
			<p class="text-sm text-red-800">{error}</p>
		</div>
	{/if}

	{#if isLoading}
		<div class="text-center py-12">
			<div
				class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-blue-600 border-r-transparent"
			></div>
			<p class="mt-2 text-gray-600">Loading users...</p>
		</div>
	{:else}
		<div class="bg-white shadow-md rounded-lg overflow-hidden">
			<table class="min-w-full divide-y divide-gray-200">
				<thead class="bg-gray-50">
					<tr>
						<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
							User
						</th>
						<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
							Email
						</th>
						<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
							Role
						</th>
						<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
							Created
						</th>
						<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
							Actions
						</th>
					</tr>
				</thead>
				<tbody class="bg-white divide-y divide-gray-200">
					{#each users as user}
						<tr>
							<td class="px-6 py-4 whitespace-nowrap">
								<div class="flex items-center">
									{#if user.picture_url}
										<img class="h-10 w-10 rounded-full" src={user.picture_url} alt="" />
									{/if}
									<div class="ml-4">
										<div class="text-sm font-medium text-gray-900">{user.name}</div>
									</div>
								</div>
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								<div class="text-sm text-gray-900">{user.email}</div>
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								<span
									class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full"
									class:bg-green-100={user.role === 'admin'}
									class:text-green-800={user.role === 'admin'}
									class:bg-gray-100={user.role === 'user'}
									class:text-gray-800={user.role === 'user'}
								>
									{user.role}
								</span>
							</td>
							<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
								{new Date(user.created_at).toLocaleDateString()}
							</td>
							<td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
								{#if user.id === authStore.user?.id}
									<span class="text-gray-400">You</span>
								{:else if updatingUserId === user.id}
									<span class="text-gray-400">Updating...</span>
								{:else}
									<select
										value={user.role}
										onchange={(e) => updateUserRole(user.id, e.currentTarget.value)}
										class="text-sm border border-gray-300 rounded px-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
									>
										<option value="user">User</option>
										<option value="admin">Admin</option>
									</select>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>

			{#if users.length === 0}
				<div class="text-center py-12">
					<p class="text-gray-500">No users found</p>
				</div>
			{/if}
		</div>
	{/if}
</div>
