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

	type DatePreset = 'today' | 'yesterday' | 'past7' | 'custom';

	function datesForPreset(preset: DatePreset): { from: string; to: string } {
		const today = new Date();
		if (preset === 'yesterday') {
			const y = new Date();
			y.setDate(y.getDate() - 1);
			const d = formatDateISO(y);
			return { from: d, to: d };
		}
		if (preset === 'past7') {
			const weekAgo = new Date();
			weekAgo.setDate(weekAgo.getDate() - 6);
			return { from: formatDateISO(weekAgo), to: formatDateISO(today) };
		}
		const d = formatDateISO(today);
		return { from: d, to: d };
	}

	let activePreset: DatePreset = $state('today');
	const now = new Date();
	let fromDate = $state(formatDateISO(now));
	let toDate = $state(formatDateISO(now));

	let selectedTypes: Set<string> = $state(new Set());

	function selectPreset(preset: DatePreset): void {
		activePreset = preset;
		const { from, to } = datesForPreset(preset);
		fromDate = from;
		toDate = to;
	}

	function onDateInputChange(): void {
		activePreset = 'custom';
	}

	function toggleType(key: string): void {
		const next = new Set(selectedTypes);
		if (next.has(key)) {
			next.delete(key);
		} else {
			next.add(key);
		}
		selectedTypes = next;
	}

	function selectAllTypes(): void {
		selectedTypes = new Set();
	}

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

	const SOURCE_TYPE_TO_ENDPOINT: Record<string, string> = {
		urine: 'urine',
		stool: 'stools',
		feeding: 'feedings'
	};

	async function handleDelete(logType: LogTypeConfig, id: string): Promise<void> {
		try {
			const typedEntry = entries.find((e) => e.entry.id === id);
			const entry = typedEntry?.entry;

			// If deleting a linked fluid_log, redirect to the source endpoint
			if (logType.key === 'fluid' && entry?.source_type && entry?.source_id) {
				const sourceEndpoint = SOURCE_TYPE_TO_ENDPOINT[entry.source_type as string];
				if (sourceEndpoint) {
					await apiClient.del(`/babies/${babyId}/${sourceEndpoint}/${entry.source_id}`);
					entries = entries.filter(
						(e) => e.entry.id !== id && e.entry.id !== entry.source_id
					);
					return;
				}
			}

			await apiClient.del(`/babies/${babyId}/${logType.endpoint}/${id}`);

			// Remove the deleted entry AND any linked fluid_log entries
			entries = entries.filter(
				(e) => e.entry.id !== id && !(e.logType.key === 'fluid' && e.entry.source_id === id)
			);
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
		const filtered = selectedTypes.size === 0
			? entries
			: entries.filter((te) => selectedTypes.has(te.logType.key));
		const groups: TypeGroup[] = [];
		const seen = new Map<string, TypeGroup>();
		for (const te of filtered) {
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
	<div class="filter-bar">
		<div class="date-presets">
			<button type="button" class={activePreset === 'today' ? 'active' : ''} onclick={() => selectPreset('today')}>Today</button>
			<button type="button" class={activePreset === 'yesterday' ? 'active' : ''} onclick={() => selectPreset('yesterday')}>Yesterday</button>
			<button type="button" class={activePreset === 'past7' ? 'active' : ''} onclick={() => selectPreset('past7')}>Past 7 Days</button>
		</div>
		<div class="date-filters">
			<input type="date" bind:value={fromDate} aria-label="From date" oninput={onDateInputChange} />
			<input type="date" bind:value={toDate} aria-label="To date" oninput={onDateInputChange} />
		</div>
		<div class="type-filter">
			<button type="button" class={selectedTypes.size === 0 ? 'active' : ''} onclick={selectAllTypes}>All</button>
			{#each LOG_TYPES as lt (lt.key)}
				<button type="button" class={selectedTypes.has(lt.key) ? 'active' : ''} onclick={() => toggleType(lt.key)}>{lt.label}</button>
			{/each}
		</div>
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

	.filter-bar {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}

	.date-presets,
	.type-filter {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}

	.date-presets button,
	.type-filter button {
		min-height: 36px;
		padding: var(--space-1) var(--space-3);
		border-radius: var(--radius-full);
		font-size: var(--font-size-sm);
		background: var(--color-surface);
		border: 1.5px solid var(--color-border);
		color: var(--color-text);
	}

	.date-presets button:hover,
	.type-filter button:hover {
		border-color: var(--color-primary);
		background: var(--color-primary-light);
	}

	.date-presets button.active,
	.type-filter button.active {
		background: var(--color-primary);
		color: var(--color-text-inverse);
		border-color: var(--color-primary);
	}

	.date-filters {
		display: flex;
		gap: var(--space-2);
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
