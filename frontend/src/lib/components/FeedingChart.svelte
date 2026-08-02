<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';

	interface FeedingDailyDataPoint {
		date: string;
		total_volume_ml: number;
		liquid_volume_ml?: number;
		solid_volume_ml?: number;
		solid_amount_g?: number;
		total_calories: number;
		feed_count: number;
		by_type: Record<string, number>;
	}

	interface Props {
		data: FeedingDailyDataPoint[];
	}

	let { data }: Props = $props();

	let config = $derived<ChartConfiguration>({
		type: 'bar',
		data: {
			labels: data.map((d) => d.date),
			datasets: [
				{
					label: 'Daily Calories (kcal)',
					data: data.map((d) => d.total_calories),
					backgroundColor: '#f59e0b',
					borderColor: '#d97706',
					borderWidth: 1,
					yAxisID: 'y'
				},
				{
					label: 'Liquid Volume (mL)',
					data: data.map((d) => d.liquid_volume_ml ?? 0),
					backgroundColor: '#3b82f6',
					borderColor: '#2563eb',
					borderWidth: 1,
					yAxisID: 'y1'
				},
				{
					label: 'Solid Volume (mL)',
					data: data.map((d) => d.solid_volume_ml ?? 0),
					backgroundColor: '#14b8a6',
					borderColor: '#0d9488',
					borderWidth: 1,
					yAxisID: 'y1'
				},
				{
					label: 'Solid Food (g)',
					data: data.map((d) => d.solid_amount_g ?? 0),
					backgroundColor: '#a855f7',
					borderColor: '#9333ea',
					borderWidth: 1,
					yAxisID: 'y1'
				}
			]
		},
		options: {
			responsive: true,
			scales: {
				x: {
					title: { display: true, text: 'Date' }
				},
				y: {
					type: 'linear',
					position: 'left',
					title: { display: true, text: 'Calories (kcal)' },
					beginAtZero: true
				},
				y1: {
					type: 'linear',
					position: 'right',
					title: { display: true, text: 'Volume (mL) / Weight (g)' },
					beginAtZero: true,
					grid: { drawOnChartArea: false }
				}
			}
		}
	});
</script>

<ChartWrapper {config} isEmpty={data.length === 0} />
