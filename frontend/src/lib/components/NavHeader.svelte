<script lang="ts">
	import { page } from '$app/stores';
	import BabySelector from './BabySelector.svelte';
	import { babies, activeBaby, setActiveBaby } from '$lib/stores/baby';
	import { currentUser } from '$lib/stores/user';

	let currentPath = $derived($page.url.pathname);
</script>

{#if $currentUser}
	<!-- Top bar: baby selector -->
	{#if $babies.length > 0}
		<div class="top-bar">
			<BabySelector
				babies={$babies}
				activeBabyId={$activeBaby?.id ?? null}
				onselect={setActiveBaby}
			/>
		</div>
	{/if}

	<!-- Bottom tab bar -->
	<nav class="bottom-nav">
		<a href="/" class="nav-tab" class:active={currentPath === '/'}>
			<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
			<span>Home</span>
		</a>

		{#if $babies.length > 0}
			<a href="/trends" class="nav-tab" class:active={currentPath === '/trends'}>
				<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
				<span>Trends</span>
			</a>

			<a href="/log" class="nav-tab log-tab" class:active={currentPath.startsWith('/log')}>
				<div class="log-btn">
					<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
				</div>
				<span>Log</span>
			</a>

			<a href="/medications" class="nav-tab" class:active={currentPath === '/medications'}>
				<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="9" y1="21" x2="9" y2="9"/></svg>
				<span>Meds</span>
			</a>

			<a href="/logs" class="nav-tab" class:active={currentPath === '/logs'}>
				<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
				<span>Logs</span>
			</a>
		{/if}

		<a href="/settings" class="nav-tab" class:active={currentPath === '/settings'}>
			<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
			<span>Settings</span>
		</a>
	</nav>
{/if}

<style>
	.top-bar {
		position: sticky;
		top: 0;
		z-index: 50;
		background: var(--color-surface);
		border-bottom: 1px solid var(--color-border);
		padding: var(--space-2) var(--space-4);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.bottom-nav {
		position: fixed;
		bottom: 0;
		left: 0;
		right: 0;
		height: var(--nav-height);
		background: var(--color-surface);
		border-top: 1px solid var(--color-border);
		display: flex;
		align-items: center;
		justify-content: space-around;
		padding-bottom: env(safe-area-inset-bottom, 0);
		z-index: 100;
		box-shadow: 0 -1px 4px rgba(0, 0, 0, 0.06);
	}

	.nav-tab {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
		font-size: var(--font-size-xs);
		font-weight: 500;
		color: var(--color-text-muted);
		background: none;
		border: none;
		padding: var(--space-1);
		min-width: var(--touch-target);
		min-height: var(--touch-target);
		text-decoration: none;
		transition: color 0.15s;
	}

	.nav-tab:hover {
		text-decoration: none;
		color: var(--color-primary);
	}

	.nav-tab.active {
		color: var(--color-primary);
	}

	.log-tab {
		position: relative;
	}

	.log-btn {
		width: 44px;
		height: 44px;
		border-radius: 50%;
		background: var(--color-primary);
		color: var(--color-text-inverse);
		display: flex;
		align-items: center;
		justify-content: center;
		margin-top: -14px;
		box-shadow: var(--shadow-md);
		transition: background-color 0.15s;
	}

	.log-tab:hover .log-btn,
	.log-tab.active .log-btn {
		background: var(--color-primary-dark);
	}

	.log-tab span {
		margin-top: -2px;
	}
</style>
