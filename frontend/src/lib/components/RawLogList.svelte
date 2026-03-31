<script lang="ts">
	import { apiClient } from '$lib/api';
	import LogEntryCard from '$lib/components/LogEntryCard.svelte';
	import type { LogTypeConfig } from '$lib/types/logs';
	import { formatDateISO } from '$lib/datetime';

	interface Props {
		babyId: string;
		logType: LogTypeConfig;
	}

	let { babyId, logType }: Props = $props();

	let entries: Record<string, unknown>[] = $state([]);
	let nextCursor: string | null = $state(null);
	let loading = $state(false);
	let error: string | null = $state(null);

	const now = new Date();
	const sevenDaysAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
	let fromDate = $state(formatDateISO(sevenDaysAgo));
	let toDate = $state(formatDateISO(now));

	interface ApiResponse {
		data: Record<string, unknown>[];
		next_cursor: string | null;
	}

	async function fetchEntries(cursor?: string): Promise<void> {
		loading = true;
		error = null;

		try {
			let url = `/babies/${babyId}/${logType.endpoint}?from=${fromDate}&to=${toDate}`;
			if (cursor) {
				url += `&cursor=${cursor}`;
			}
			const response = await apiClient.get<ApiResponse>(url);
			if (cursor) {
				entries = [...entries, ...response.data];
			} else {
				entries = response.data;
			}
			nextCursor = response.next_cursor;
		} catch (e) {
			error = e instanceof Error ? e.message : 'An error occurred';
		} finally {
			loading = false;
		}
	}

	function handleLoadMore(): void {
		if (nextCursor) {
			fetchEntries(nextCursor);
		}
	}

	async function handleDelete(id: string): Promise<void> {
		try {
			await apiClient.del(`/babies/${babyId}/${logType.endpoint}/${id}`);
			entries = entries.filter((e) => e.id !== id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete entry';
		}
	}

	$effect(() => {
		// Track reactive dependencies
		babyId;
		logType;
		fromDate;
		toDate;
		fetchEntries();
	});
</script>

<div class="log-list">
	<div class="date-filters">
		<input type="date" bind:value={fromDate} aria-label="From date" />
		<input type="date" bind:value={toDate} aria-label="To date" />
	</div>

	{#if loading && entries.length === 0}
		<div class="loading">Loading...</div>
	{:else if error}
		<div class="error">{error}</div>
	{:else if entries.length === 0}
		<div class="empty">No entries found.</div>
	{:else}
		{#each entries as entry (entry.id)}
			<LogEntryCard {entry} {logType} ondelete={handleDelete} />
		{/each}

		{#if loading}
			<div class="loading">Loading...</div>
		{/if}

		{#if nextCursor}
			<button class="load-more" onclick={handleLoadMore}>Load More</button>
		{/if}
	{/if}
</div>

<style>
	.log-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.date-filters {
		display: flex;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}

	.date-filters input {
		flex: 1;
	}

	.load-more {
		width: 100%;
		margin-top: var(--space-3);
	}
</style>
