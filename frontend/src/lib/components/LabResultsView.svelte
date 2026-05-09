<script lang="ts">
	import { untrack } from 'svelte';
	import '$lib/chart-setup';
	import { apiClient } from '$lib/api';
	import { testColorMap } from '$lib/chart-utils';
	import { formatDateISO } from '$lib/datetime';
	import type { LabResult, LabResultsPage } from '$lib/types/lab';
	import type { ImagingStudy, ImagingStudiesPage } from '$lib/types/imaging';
	import DateRangeSelector from './DateRangeSelector.svelte';
	import TestFilter from './TestFilter.svelte';
	import LabTrendsChart from './LabTrendsChart.svelte';
	import LabDateGroup from './LabDateGroup.svelte';
	import ImagingStudyCard from './ImagingStudyCard.svelte';

	interface Props {
		babyId: string;
	}

	let { babyId }: Props = $props();

	let allResults = $state<LabResult[]>([]);
	let imagingStudies = $state<ImagingStudy[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let selectedRange = $state('30d');
	let selectedTests = $state<Set<string>>(new Set());

	function computeDateRange(range: string, customFrom?: string, customTo?: string): { from: string; to: string } {
		const now = new Date();
		const toStr = formatDateISO(now);

		if (range === 'custom' && customFrom && customTo) {
			return { from: customFrom, to: customTo };
		}

		const daysMap: Record<string, number> = { '7d': 7, '14d': 14, '30d': 30, '90d': 90 };
		const days = daysMap[range] ?? 30;
		const from = new Date(now);
		from.setDate(from.getDate() - days);
		return { from: formatDateISO(from), to: toStr };
	}

	async function fetchAllPages(range: string, cFrom?: string, cTo?: string): Promise<void> {
		loading = true;
		error = null;
		try {
			const { from, to } = computeDateRange(range, cFrom, cTo);

			// Fetch numeric labs and imaging studies in parallel; each paginates independently.
			const [labResults, studies] = await Promise.all([
				fetchAllLabs(from, to),
				fetchAllImagingStudies(from, to)
			]);

			allResults = labResults;
			imagingStudies = studies;
		} catch {
			error = 'Failed to load lab results';
		} finally {
			loading = false;
		}
	}

	async function fetchAllLabs(from: string, to: string): Promise<LabResult[]> {
		const accumulated: LabResult[] = [];
		let cursor: string | null = null;
		do {
			const params = new URLSearchParams({ from, to });
			if (cursor) params.set('cursor', cursor);
			const page = await apiClient.get<LabResultsPage>(`/babies/${babyId}/labs?${params}`);
			accumulated.push(...page.data);
			cursor = page.next_cursor;
		} while (cursor);
		return accumulated;
	}

	async function fetchAllImagingStudies(from: string, to: string): Promise<ImagingStudy[]> {
		const accumulated: ImagingStudy[] = [];
		let cursor: string | null = null;
		try {
			do {
				const params = new URLSearchParams({ from, to });
				if (cursor) params.set('cursor', cursor);
				const page = await apiClient.get<ImagingStudiesPage>(
					`/babies/${babyId}/imaging-studies?${params}`
				);
				accumulated.push(...page.data);
				cursor = page.next_cursor;
			} while (cursor);
		} catch {
			// Imaging studies are optional in mixed view; non-fatal
		}
		return accumulated;
	}

	function handleRangeChange(range: string, cFrom?: string, cTo?: string): void {
		selectedRange = range;
		void fetchAllPages(range, cFrom, cTo);
	}

	function handleTestFilterChange(tests: Set<string>): void {
		selectedTests = tests;
	}

	let availableTests = $derived(
		Array.from(new Set(allResults.map((r) => r.test_name))).sort()
	);

	let testColors = $derived(testColorMap(availableTests));

	let filteredResults = $derived(
		selectedTests.size === 0
			? allResults
			: allResults.filter((r) => selectedTests.has(r.test_name))
	);

	let chartData = $derived.by(() => {
		const grouped: Record<string, { timestamp: string; test_name: string; value: string; unit: string }[]> = {};
		for (const r of filteredResults) {
			if (!grouped[r.test_name]) grouped[r.test_name] = [];
			grouped[r.test_name].push({
				timestamp: r.timestamp,
				test_name: r.test_name,
				value: r.value,
				unit: r.unit ?? ''
			});
		}
		return grouped;
	});

	let dateGroups = $derived.by(() => {
		const groups = new Map<string, { results: LabResult[]; studies: ImagingStudy[] }>();
		for (const r of filteredResults) {
			const date = formatDateISO(new Date(r.timestamp));
			if (!groups.has(date)) groups.set(date, { results: [], studies: [] });
			groups.get(date)!.results.push(r);
		}
		for (const s of imagingStudies) {
			const date = s.study_date;
			if (!groups.has(date)) groups.set(date, { results: [], studies: [] });
			groups.get(date)!.studies.push(s);
		}
		return Array.from(groups.entries())
			.sort((a, b) => b[0].localeCompare(a[0]))
			.map(([date, group]) => ({
				date,
				results: group.results.sort((a, b) => a.test_name.localeCompare(b.test_name)),
				studies: group.studies
			}));
	});

	$effect(() => {
		void babyId;
		untrack(() => { void fetchAllPages(selectedRange); });
	});
</script>

<div class="lab-results-view">
	<div class="actions">
		<a href="/log/imaging" class="add-imaging-btn">+ Add imaging study</a>
	</div>

	<DateRangeSelector {selectedRange} onchange={handleRangeChange} />

	{#if loading}
		<div class="loading">Loading...</div>
	{:else if error}
		<div class="error">{error}</div>
	{:else if allResults.length === 0 && imagingStudies.length === 0}
		<div class="empty">No lab results found for this period.</div>
	{:else}
		{#if availableTests.length > 0}
			<TestFilter tests={availableTests} selected={selectedTests} onchange={handleTestFilterChange} colors={testColors} />

			<section class="chart-section">
				<h3>Lab Trends</h3>
				<LabTrendsChart data={chartData} colors={testColors} />
			</section>
		{/if}

		<div class="date-groups">
			{#each dateGroups as group (group.date)}
				{#if group.results.length > 0}
					<LabDateGroup date={group.date} results={group.results} />
				{/if}
				{#each group.studies as study (study.id)}
					<ImagingStudyCard {study} />
				{/each}
			{/each}
		</div>
	{/if}
</div>

<style>
	.lab-results-view {
		padding-top: var(--space-2);
	}

	.actions {
		display: flex;
		justify-content: flex-end;
		margin-bottom: var(--space-2);
	}

	.add-imaging-btn {
		padding: var(--space-1) var(--space-3);
		background: var(--color-primary, #0d6efd);
		color: white;
		border-radius: var(--radius-md, 6px);
		font-size: var(--font-size-sm, 0.875rem);
		text-decoration: none;
	}

	.chart-section {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		box-shadow: var(--shadow-sm);
		margin-bottom: var(--space-4);
	}

	.chart-section h3 {
		margin: 0 0 var(--space-3) 0;
	}

	.date-groups {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.loading, .error, .empty {
		text-align: center;
		padding: var(--space-8);
		color: var(--color-text-muted);
	}

	.error {
		color: var(--color-error);
	}
</style>
