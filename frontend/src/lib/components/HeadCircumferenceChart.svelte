<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';
	import type { Percentiles } from '$lib/types/percentiles';
	import { legendFilter, percentileSubtitle } from '$lib/chart-utils';

	interface HeadCircumferenceDataPoint {
		timestamp: string;
		circumference_cm: number;
	}

	interface Props {
		data: HeadCircumferenceDataPoint[];
		percentiles: Percentiles | null;
		dateOfBirth: string;
	}

	let { data, percentiles, dateOfBirth }: Props = $props();

	function ageDaysForTimestamp(timestamp: string): number {
		const birth = new Date(dateOfBirth).getTime();
		const ts = new Date(timestamp).getTime();
		return Math.floor((ts - birth) / (24 * 60 * 60 * 1000));
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
			data: points.map((p) => ({ x: p.age_days, y: p.value ?? 0 })),
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
					label: 'Head Circumference',
					data: data.map((d) => ({
						x: ageDaysForTimestamp(d.timestamp),
						y: d.circumference_cm
					})),
					borderColor: '#ec4899',
					backgroundColor: '#ec489980',
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
				subtitle: percentileSubtitle
			},
			scales: {
				x: {
					type: 'linear',
					title: { display: true, text: 'Age (days)' }
				},
				y: {
					title: { display: true, text: 'Head Circumference (cm)' }
				}
			}
		}
	});
</script>

<ChartWrapper {config} isEmpty={data.length === 0} />
