<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';

	interface DiaperDailyDataPoint {
		date: string;
		wet_count: number;
		stool_count: number;
	}

	interface Props {
		data: DiaperDailyDataPoint[];
	}

	let { data }: Props = $props();

	let config = $derived<ChartConfiguration>({
		type: 'bar',
		data: {
			labels: data.map((d) => d.date),
			datasets: [
				{
					label: 'Wet',
					data: data.map((d) => d.wet_count),
					backgroundColor: '#3b82f6',
					borderColor: '#2563eb',
					borderWidth: 1
				},
				{
					label: 'Stool',
					data: data.map((d) => d.stool_count),
					backgroundColor: '#a16207',
					borderColor: '#854d0e',
					borderWidth: 1
				}
			]
		},
		options: {
			responsive: true,
			scales: {
				x: {
					stacked: true,
					title: { display: true, text: 'Date' }
				},
				y: {
					stacked: true,
					title: { display: true, text: 'Count' },
					beginAtZero: true
				}
			}
		}
	});
</script>

<ChartWrapper {config} isEmpty={data.length === 0} />
