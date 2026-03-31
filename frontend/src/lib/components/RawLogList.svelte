<script lang="ts">
	import { apiClient } from '$lib/api';
	import LogEntryRow from '$lib/components/LogEntryRow.svelte';
	import { LOG_TYPES, type LogTypeConfig } from '$lib/types/logs';
	import { formatDateISO } from '$lib/datetime';

	interface Props {
		babyId: string;
	}

	let { babyId }: Props = $props();

	interface TypedEntry {
		entry: Record<string, unknown>;
		logType: LogTypeConfig;
	}

	let entries: TypedEntry[] = $state([]);
	let loading = $state(false);
	let error: string | null = $state(null);
	let medNames: Record<string, string> = $state({});

	const now = new Date();
	let fromDate = $state(formatDateISO(now));
	let toDate = $state(formatDateISO(now));

	interface ApiResponse {
		data: Record<string, unknown>[];
		next_cursor: string | null;
	}

	function entryTimestamp(entry: Record<string, unknown>): string {
		return (entry.timestamp ?? entry.given_at ?? entry.created_at ?? '') as string;
	}

	async function fetchAllEntries(): Promise<void> {
		loading = true;
		error = null;

		try {
			const results = await Promise.all(
				LOG_TYPES.map(async (lt) => {
					const url = `/babies/${babyId}/${lt.endpoint}?from=${fromDate}&to=${toDate}`;
					const response = await apiClient.get<ApiResponse>(url);
					return response.data.map((entry) => ({ entry, logType: lt }));
				})
			);

			const merged = results.flat();
			merged.sort((a, b) => {
				const ta = entryTimestamp(a.entry);
				const tb = entryTimestamp(b.entry);
				return ta.localeCompare(tb); // oldest first
			});

			entries = merged;
		} catch (e) {
			error = e instanceof Error ? e.message : 'An error occurred';
		} finally {
			loading = false;
		}
	}

	async function handleDelete(logType: LogTypeConfig, id: string): Promise<void> {
		try {
			await apiClient.del(`/babies/${babyId}/${logType.endpoint}/${id}`);
			entries = entries.filter((e) => e.entry.id !== id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete entry';
		}
	}

	async function fetchMedNames(): Promise<void> {
		try {
			const data = await apiClient.get<{ medications?: { id: string; name: string; dose: string }[] } | { id: string; name: string; dose: string }[]>(`/babies/${babyId}/medications`);
			const meds = Array.isArray(data) ? data : (data.medications ?? []);
			const map: Record<string, string> = {};
			for (const m of meds) {
				map[m.id] = `${m.name} ${m.dose}`;
			}
			medNames = map;
		} catch {
			// Non-critical
		}
	}

	interface TypeGroup {
		logType: LogTypeConfig;
		items: TypedEntry[];
	}

	let groupedEntries: TypeGroup[] = $derived.by(() => {
		const groups: TypeGroup[] = [];
		const seen = new Map<string, TypeGroup>();
		for (const te of entries) {
			let group = seen.get(te.logType.key);
			if (!group) {
				group = { logType: te.logType, items: [] };
				seen.set(te.logType.key, group);
				groups.push(group);
			}
			group.items.push(te);
		}
		return groups;
	});

	let feedingTotalMl = $derived(
		entries
			.filter((e) => e.logType.key === 'feeding')
			.reduce((sum, e) => sum + (Number(e.entry.volume_ml) || 0), 0)
	);

	let fluidOutputTotalMl = $derived(
		entries
			.filter((e) => e.logType.key === 'fluid' && e.entry.direction === 'output')
			.reduce((sum, e) => sum + (Number(e.entry.volume_ml) || 0), 0)
	);

	$effect(() => {
		babyId;
		fromDate;
		toDate;
		fetchAllEntries();
		fetchMedNames();
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
		{#if feedingTotalMl > 0 || fluidOutputTotalMl > 0}
			<div class="totals">
				{#if feedingTotalMl > 0}
					<span class="total-item">Feeding: {feedingTotalMl} mL</span>
				{/if}
				{#if fluidOutputTotalMl > 0}
					<span class="total-item">Output: {fluidOutputTotalMl} mL</span>
				{/if}
			</div>
		{/if}
		{#each groupedEntries as group (group.logType.key)}
			<h2 class="type-heading">{group.logType.label}</h2>
			<div class="entry-list">
				{#each group.items as { entry, logType } (entry.id)}
					<LogEntryRow {entry} {logType} ondelete={(id) => handleDelete(logType, id)} {medNames} />
				{/each}
			</div>
		{/each}

		{#if loading}
			<div class="loading">Loading...</div>
		{/if}
	{/if}
</div>

<style>
	.log-list {
		display: flex;
		flex-direction: column;
	}

	.date-filters {
		display: flex;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}

	.date-filters input {
		flex: 1;
	}

	.type-heading {
		font-size: var(--font-size-sm);
		font-weight: 600;
		color: var(--color-text-muted);
		margin: var(--space-3) 0 0 0;
		padding: var(--space-1) var(--space-3);
	}

	.entry-list {
		border-top: 1px solid var(--color-border);
	}

	.totals {
		display: flex;
		gap: var(--space-4);
		padding: var(--space-2) var(--space-3);
		margin-bottom: var(--space-2);
		font-size: var(--font-size-sm);
		font-weight: 600;
		color: var(--color-text-muted);
	}
</style>
