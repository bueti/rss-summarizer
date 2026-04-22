<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';

	const navItems = [
		{ href: '/', label: 'Dashboard' },
		{ href: '/saved', label: 'Saved' },
		{ href: '/archive', label: 'Archive' },
		{ href: '/feeds', label: 'Feeds' },
		{ href: '/email-sources', label: 'Email Sources' },
		{ href: '/topics', label: 'Topics' },
		{ href: '/monitoring', label: 'Monitoring' }
	];

	const adminNavItems = [
		{ href: '/admin/users', label: 'Users' },
		{ href: '/admin/llm', label: 'LLM Config' }
	];

	// Public routes that don't require authentication
	const publicRoutes = ['/login', '/auth/callback'];

	let mobileMenuOpen = $state(false);
	let userMenuOpen = $state(false);

	onMount(async () => {
		await authStore.initialize();

		// Only redirect to login if not authenticated and not on a public route
		// Note: In dev mode, the backend injects a dev user, so isAuthenticated will be true
		const currentPath = $page.url.pathname;
		if (!authStore.isAuthenticated && !publicRoutes.includes(currentPath)) {
			goto('/login');
		}
	});

	// Close mobile menu when route changes
	$effect(() => {
		$page.url.pathname;
		mobileMenuOpen = false;
		userMenuOpen = false;
	});
</script>

{#if authStore.isLoading}
	<div class="min-h-screen flex items-center justify-center bg-gray-50">
		<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
	</div>
{:else if !authStore.isAuthenticated && !publicRoutes.includes($page.url.pathname)}
	<!-- Will redirect to login via onMount -->
{:else}
	<div class="min-h-screen bg-gray-50">
		{#if authStore.isAuthenticated}
			<nav class="bg-white border-b border-gray-200">
				<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
					<div class="flex justify-between h-16">
						<div class="flex">
							<div class="flex-shrink-0 flex items-center">
								<h1 class="text-xl font-bold text-gray-900">RSS Summarizer</h1>
							</div>
							<div class="hidden md:ml-6 md:flex md:space-x-8">
								{#each navItems as item}
									<a
										href={item.href}
										class="border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium transition-colors"
										class:!border-blue-500={$page.url.pathname === item.href}
										class:!text-gray-900={$page.url.pathname === item.href}
									>
										{item.label}
									</a>
								{/each}
								{#if authStore.isAdmin}
									<div class="flex items-center px-1">
										<span class="text-gray-400">|</span>
									</div>
									{#each adminNavItems as item}
										<a
											href={item.href}
											class="border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium transition-colors"
											class:!border-blue-500={$page.url.pathname === item.href}
											class:!text-gray-900={$page.url.pathname === item.href}
										>
											{item.label}
										</a>
									{/each}
								{/if}
							</div>
						</div>

						<!-- Mobile menu button -->
						<div class="flex items-center md:hidden">
							<button
								onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
								class="inline-flex items-center justify-center p-2 rounded-md text-gray-400 hover:text-gray-500 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-blue-500"
								aria-expanded={mobileMenuOpen}
							>
								<span class="sr-only">Open main menu</span>
								{#if !mobileMenuOpen}
									<svg
										class="block h-6 w-6"
										fill="none"
										viewBox="0 0 24 24"
										stroke-width="1.5"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5"
										/>
									</svg>
								{:else}
									<svg
										class="block h-6 w-6"
										fill="none"
										viewBox="0 0 24 24"
										stroke-width="1.5"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M6 18L18 6M6 6l12 12"
										/>
									</svg>
								{/if}
							</button>
						</div>

						<!-- User menu (desktop) -->
						<div class="hidden md:flex items-center relative">
							<button
								onclick={() => (userMenuOpen = !userMenuOpen)}
								class="flex items-center space-x-2 focus:outline-none hover:opacity-80 transition-opacity"
							>
								{#if authStore.user?.picture_url}
									<img
										src={authStore.user.picture_url}
										alt={authStore.user.name}
										class="h-8 w-8 rounded-full"
									/>
								{/if}
								<span class="text-sm text-gray-700">{authStore.user?.name}</span>
								<svg
									class="h-4 w-4 text-gray-400"
									fill="none"
									viewBox="0 0 24 24"
									stroke-width="1.5"
									stroke="currentColor"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M19.5 8.25l-7.5 7.5-7.5-7.5"
									/>
								</svg>
							</button>

							{#if userMenuOpen}
								<div
									class="absolute right-0 top-12 mt-2 w-48 bg-white rounded-md shadow-lg py-1 z-50 border border-gray-200"
								>
									<a
										href="/settings"
										class="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
										onclick={() => (userMenuOpen = false)}
									>
										Settings
									</a>
									<button
										onclick={() => authStore.logout()}
										class="block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
									>
										Logout
									</button>
								</div>
							{/if}
						</div>
					</div>
				</div>

				<!-- Mobile menu -->
				{#if mobileMenuOpen}
					<div class="md:hidden">
						<div class="pt-2 pb-3 space-y-1">
							{#each navItems as item}
								<a
									href={item.href}
									class="border-transparent text-gray-500 hover:bg-gray-50 hover:border-gray-300 hover:text-gray-700 block pl-3 pr-4 py-2 border-l-4 text-base font-medium transition-colors"
									class:!border-blue-500={$page.url.pathname === item.href}
									class:!text-gray-900={$page.url.pathname === item.href}
									class:!bg-blue-50={$page.url.pathname === item.href}
								>
									{item.label}
								</a>
							{/each}

							{#if authStore.isAdmin}
								<div class="border-t border-gray-200 my-2"></div>
								{#each adminNavItems as item}
									<a
										href={item.href}
										class="border-transparent text-gray-500 hover:bg-gray-50 hover:border-gray-300 hover:text-gray-700 block pl-3 pr-4 py-2 border-l-4 text-base font-medium transition-colors"
										class:!border-blue-500={$page.url.pathname === item.href}
										class:!text-gray-900={$page.url.pathname === item.href}
										class:!bg-blue-50={$page.url.pathname === item.href}
									>
										{item.label}
									</a>
								{/each}
							{/if}
						</div>

						<!-- Mobile user menu -->
						<div class="pt-4 pb-3 border-t border-gray-200">
							<div class="flex items-center px-4">
								{#if authStore.user?.picture_url}
									<div class="flex-shrink-0">
										<img
											src={authStore.user.picture_url}
											alt={authStore.user.name}
											class="h-10 w-10 rounded-full"
										/>
									</div>
								{/if}
								<div class="ml-3">
									<div class="text-base font-medium text-gray-800">{authStore.user?.name}</div>
									<div class="text-sm font-medium text-gray-500">{authStore.user?.email}</div>
								</div>
							</div>
							<div class="mt-3 space-y-1">
								<a
									href="/settings"
									class="block px-4 py-2 text-base font-medium text-gray-500 hover:text-gray-800 hover:bg-gray-100"
								>
									Settings
								</a>
								<button
									onclick={() => authStore.logout()}
									class="block w-full text-left px-4 py-2 text-base font-medium text-gray-500 hover:text-gray-800 hover:bg-gray-100"
								>
									Logout
								</button>
							</div>
						</div>
					</div>
				{/if}
			</nav>

			<main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
				<slot />
			</main>
		{:else}
			<!-- Public routes (login, callback) -->
			<slot />
		{/if}
	</div>
{/if}
