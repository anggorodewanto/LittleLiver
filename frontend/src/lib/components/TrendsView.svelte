<script lang="ts">
	import { untrack } from 'svelte';
	import '$lib/chart-setup';
	import { apiClient } from '$lib/api';
	import { formatDateISO } from '$lib/datetime';
	import type { Percentiles } from '$lib/types/percentiles';
	import DateRangeSelector from './DateRangeSelector.svelte';
	import StoolColorChart from './StoolColorChart.svelte';
	import WeightChart from './WeightChart.svelte';
	import TemperatureChart from './TemperatureChart.svelte';
	import AbdomenGirthChart from './AbdomenGirthChart.svelte';
	import FeedingChart from './FeedingChart.svelte';
	import DiaperChart from './DiaperChart.svelte';
	import LabTrendsChart from './LabTrendsChart.svelte';

	interface Props {
		babyId: string;
		sex: 'male' | 'female';
		dateOfBirth: string;
	}

	interface DashboardResponse {
		chart_data_series: {
			temperature: { timestamp: string; value: number; method: string }[];
			weight: { timestamp: string; weight_kg: number; measurement_source: string }[];
			stool_color: { timestamp: string; color_score: number }[];
			feeding_daily: { date: string; total_volume_ml: number; total_calories: number; feed_count: number; by_type: Record<string, number> }[];
			diaper_daily: { date: string; wet_count: number; stool_count: number }[];
			abdomen_girth: { timestamp: string; girth_cm: number }[];
			lab_trends: Record<string, { timestamp: string; test_name: string; value: string; unit: string }[]>;
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

	function computeDateRange(range: string, customFrom?: string, customTo?: string): { from: string; to: string } {
		const now = new Date();
		const toStr = formatDateISO(now);

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
		return { from: formatDateISO(from), to: toStr };
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
					.catch(() => null)
			]);

			dashboard = dashboardData;
			percentiles = percentileData?.percentiles ?? null;
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
	{:else if dashboard}
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

			<section class="chart-section">
				<h3>Abdomen Girth</h3>
				<AbdomenGirthChart data={dashboard.chart_data_series.abdomen_girth} />
			</section>

			<section class="chart-section">
				<h3>Feeding</h3>
				<FeedingChart data={dashboard.chart_data_series.feeding_daily} />
			</section>

			<section class="chart-section">
				<h3>Diaper Counts</h3>
				<DiaperChart data={dashboard.chart_data_series.diaper_daily} />
			</section>

			<section class="chart-section">
				<h3>Lab Trends</h3>
				<LabTrendsChart data={dashboard.chart_data_series.lab_trends} />
			</section>
		</div>
	{/if}
</div>
