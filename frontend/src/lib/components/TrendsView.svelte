<script lang="ts">
	import { untrack } from 'svelte';
	import '$lib/chart-setup';
	import { apiClient } from '$lib/api';
	import DateRangeSelector from './DateRangeSelector.svelte';
	import StoolColorChart from './StoolColorChart.svelte';
	import WeightChart from './WeightChart.svelte';
	import TemperatureChart from './TemperatureChart.svelte';

	interface Props {
		babyId: string;
		sex: 'male' | 'female';
		dateOfBirth: string;
	}

	interface PercentilePoint {
		age_days: number;
		weight_kg: number;
	}

	interface Percentiles {
		p3: PercentilePoint[];
		p15: PercentilePoint[];
		p50: PercentilePoint[];
		p85: PercentilePoint[];
		p97: PercentilePoint[];
	}

	interface DashboardResponse {
		chart_data_series: {
			temperature: { timestamp: string; value: number; method: string }[];
			weight: { timestamp: string; weight_kg: number; measurement_source: string }[];
			stool_color: { timestamp: string; color_score: number }[];
			feeding_daily: unknown[];
			diaper_daily: unknown[];
			abdomen_girth: unknown[];
			lab_trends: Record<string, unknown>;
		};
		[key: string]: unknown;
	}

	interface PercentileResponse {
		percentiles: Percentiles;
	}

	let { babyId, sex, dateOfBirth }: Props = $props();

	let selectedRange = $state('7d');
	let loading = $state(true);
	let error = $state<string | null>(null);
	let dashboard = $state<DashboardResponse | null>(null);
	let percentiles = $state<Percentiles | null>(null);

	function formatDate(date: Date): string {
		return date.toISOString().split('T')[0];
	}

	function computeDateRange(range: string, customFrom?: string, customTo?: string): { from: string; to: string } {
		const now = new Date();
		const toStr = formatDate(now);

		if (range === 'custom' && customFrom && customTo) {
			return { from: customFrom, to: customTo };
		}

		const daysMap: Record<string, number> = {
			'7d': 7,
			'14d': 14,
			'30d': 30,
			'90d': 90
		};
		const days = daysMap[range] ?? 7;
		// eslint-disable-next-line svelte/prefer-svelte-reactivity -- used in non-reactive function
		const from = new Date(now);
		from.setDate(from.getDate() - days);
		return { from: formatDate(from), to: toStr };
	}

	async function fetchData(range: string, cFrom?: string, cTo?: string): Promise<void> {
		loading = true;
		error = null;
		try {
			const { from, to } = computeDateRange(range, cFrom, cTo);

			// Compute age range for WHO percentiles
			const birthDate = new Date(dateOfBirth);
			const fromDate = new Date(from);
			const toDate = new Date(to);
			const fromDays = Math.max(0, Math.floor((fromDate.getTime() - birthDate.getTime()) / (24 * 60 * 60 * 1000)));
			const toDays = Math.max(0, Math.floor((toDate.getTime() - birthDate.getTime()) / (24 * 60 * 60 * 1000)));

			const [dashboardData, percentileData] = await Promise.all([
				apiClient.get<DashboardResponse>(`/babies/${babyId}/dashboard?from=${from}&to=${to}`),
				apiClient.get<PercentileResponse>(`/who/percentiles?sex=${sex}&from_days=${fromDays}&to_days=${toDays}`)
			]);

			dashboard = dashboardData;
			percentiles = percentileData.percentiles;
		} catch {
			error = 'Failed to load trends data';
		} finally {
			loading = false;
		}
	}

	function handleRangeChange(range: string, cFrom?: string, cTo?: string): void {
		selectedRange = range;
		void fetchData(range, cFrom, cTo);
	}

	$effect(() => {
		void babyId;
		untrack(() => { void fetchData(selectedRange); });
	});
</script>

<div class="trends-view">
	<DateRangeSelector {selectedRange} onchange={handleRangeChange} />

	{#if loading}
		<div class="loading">Loading...</div>
	{:else if error}
		<div class="error">{error}</div>
	{:else if dashboard && percentiles}
		<div class="charts">
			<section class="chart-section">
				<h3>Stool Color</h3>
				<StoolColorChart data={dashboard.chart_data_series.stool_color} />
			</section>

			<section class="chart-section">
				<h3>Weight</h3>
				<WeightChart
					data={dashboard.chart_data_series.weight}
					{percentiles}
					{dateOfBirth}
				/>
			</section>

			<section class="chart-section">
				<h3>Temperature</h3>
				<TemperatureChart data={dashboard.chart_data_series.temperature} />
			</section>
		</div>
	{/if}
</div>
