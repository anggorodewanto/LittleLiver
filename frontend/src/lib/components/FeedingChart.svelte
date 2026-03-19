<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';

	interface FeedingDailyDataPoint {
		date: string;
		total_volume_ml: number;
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
					label: 'Daily Calories',
					data: data.map((d) => d.total_calories),
					backgroundColor: '#f59e0b',
					borderColor: '#d97706',
					borderWidth: 1
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
					title: { display: true, text: 'Calories (kcal)' },
					beginAtZero: true
				}
			}
		}
	});
</script>

<ChartWrapper {config} isEmpty={data.length === 0} />
