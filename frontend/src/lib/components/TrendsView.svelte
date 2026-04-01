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
	import HeadCircumferenceChart from './HeadCircumferenceChart.svelte';
	import UpperArmCircumferenceChart from './UpperArmCircumferenceChart.svelte';

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
			head_circumference: { timestamp: string; circumference_cm: number }[];
			upper_arm_circumference: { timestamp: string; circumference_cm: number }[];
			lab_trends: Record<string, { timestamp: string; test_name: string; value: string; unit: string }[]>;
		};
		[key: string]: unknown;
	}

	interface CurvePoint {
		age_days: number;
		weight_kg: number;
	}

	interface PercentileCurve {
		percentile: number;
		points: CurvePoint[];
	}

	interface PercentileResponse {
		curves: PercentileCurve[];
	}

	function transformCurves(curves: PercentileCurve[]): Percentiles {
		const keyMap: Record<number, keyof Percentiles> = { 3: 'p3', 15: 'p15', 50: 'p50', 85: 'p85', 97: 'p97' };
		const result: Percentiles = { p3: [], p15: [], p50: [], p85: [], p97: [] };
		for (const curve of curves) {
			const key = keyMap[curve.percentile];
			if (key) {
				result[key] = curve.points.map(p => ({ age_days: p.age_days, value: p.weight_kg }));
			}
		}
		return result;
	}

	let { babyId, sex, dateOfBirth }: Props = $props();

	let selectedRange = $state('7d');
	let loading = $state(true);
	let error = $state<string | null>(null);
	let dashboard = $state<DashboardResponse | null>(null);
	let percentiles = $state<Percentiles | null>(null);
	let hcPercentiles = $state<Percentiles | null>(null);

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

			const [dashboardData, percentileData, hcPercentileData] = await Promise.all([
				apiClient.get<DashboardResponse>(`/babies/${babyId}/dashboard?from=${from}&to=${to}`),
				apiClient.get<PercentileResponse>(`/who/percentiles?sex=${sex}&from_days=${fromDays}&to_days=${toDays}`)
					.catch(() => null),
				apiClient.get<PercentileResponse>(`/who/percentiles?sex=${sex}&metric=head_circumference&from_days=${fromDays}&to_days=${toDays}`)
					.catch(() => null)
			]);

			dashboard = dashboardData;
			percentiles = percentileData?.curves ? transformCurves(percentileData.curves) : null;
			hcPercentiles = hcPercentileData?.curves ? transformCurves(hcPercentileData.curves) : null;
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
				<h3>Head Circumference</h3>
				<HeadCircumferenceChart
					data={dashboard.chart_data_series.head_circumference}
					percentiles={hcPercentiles}
					{dateOfBirth}
				/>
			</section>

			<section class="chart-section">
				<h3>Upper Arm Circumference</h3>
				<UpperArmCircumferenceChart data={dashboard.chart_data_series.upper_arm_circumference} />
			</section>

			<section class="chart-section">
				<h3>Lab Trends</h3>
				<LabTrendsChart data={dashboard.chart_data_series.lab_trends} />
			</section>
		</div>
	{/if}
</div>

<style>
	.trends-view {
		padding-top: var(--space-2);
	}

	.charts {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.chart-section {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		box-shadow: var(--shadow-sm);
	}
</style>
