<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';
	import type { Percentiles } from '$lib/types/percentiles';
	import {
		legendFilter,
		percentileSubtitle,
		dateTickCallback,
		dateTooltipTitle
	} from '$lib/chart-utils';

	interface HeightDataPoint {
		timestamp: string;
		height_cm: number;
		measurement_source?: string | null;
	}

	interface Props {
		data: HeightDataPoint[];
		percentiles: Percentiles | null;
		dateOfBirth: string;
	}

	let { data, percentiles, dateOfBirth }: Props = $props();

	const DAY_MS = 24 * 60 * 60 * 1000;

	function dateMsForAgeDays(ageDays: number): number {
		return new Date(dateOfBirth).getTime() + ageDays * DAY_MS;
	}

	const PERCENTILE_COLORS: Record<string, string> = {
		'3rd': '#ef444480',
		'15th': '#f97316a0',
		'50th': '#22c55ea0',
		'85th': '#f97316a0',
		'97th': '#ef444480'
	};

	function buildPercentileDatasets() {
		if (!percentiles) return [];
		return (
			[
				['3rd', 'p3', percentiles.p3],
				['15th', 'p15', percentiles.p15],
				['50th', 'p50', percentiles.p50],
				['85th', 'p85', percentiles.p85],
				['97th', 'p97', percentiles.p97]
			] as [string, string, { age_days: number; value?: number }[]][]
		).map(([label, , points]) => ({
			label,
			data: points.map((p) => ({ x: dateMsForAgeDays(p.age_days), y: p.value ?? 0 })),
			borderColor: PERCENTILE_COLORS[label],
			borderDash: [5, 5],
			borderWidth: 1,
			pointRadius: 0,
			fill: false
		}));
	}

	let config = $derived<ChartConfiguration>({
		type: 'line',
		data: {
			datasets: [
				{
					label: 'Height',
					data: data.map((d) => ({
						x: new Date(d.timestamp).getTime(),
						y: d.height_cm
					})),
					borderColor: '#8b5cf6',
					backgroundColor: '#8b5cf680',
					borderWidth: 2,
					pointRadius: 4,
					fill: false
				},
				...buildPercentileDatasets()
			]
		},
		options: {
			responsive: true,
			plugins: {
				legend: { labels: { filter: legendFilter } },
				subtitle: percentileSubtitle,
				tooltip: { callbacks: { title: dateTooltipTitle } }
			},
			scales: {
				x: {
					type: 'linear',
					title: { display: true, text: 'Date' },
					ticks: {
						callback: dateTickCallback
					}
				},
				y: {
					title: { display: true, text: 'Height (cm)' }
				}
			}
		}
	});
</script>

<ChartWrapper {config} isEmpty={data.length === 0} />
