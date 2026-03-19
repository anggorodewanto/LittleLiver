<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';
	import { dateTickCallback } from '$lib/chart-utils';

	interface AbdomenGirthDataPoint {
		timestamp: string;
		girth_cm: number;
	}

	interface Props {
		data: AbdomenGirthDataPoint[];
	}

	let { data }: Props = $props();

	let config = $derived<ChartConfiguration>({
		type: 'line',
		data: {
			datasets: [
				{
					label: 'Abdomen Girth',
					data: data.map((d) => ({
						x: new Date(d.timestamp).getTime(),
						y: d.girth_cm
					})),
					borderColor: '#8b5cf6',
					backgroundColor: '#8b5cf680',
					borderWidth: 2,
					pointRadius: 4,
					fill: false
				}
			]
		},
		options: {
			responsive: true,
			scales: {
				x: {
					type: 'linear',
					title: { display: true, text: 'Date' },
					ticks: {
						callback: dateTickCallback
					}
				},
				y: {
					title: { display: true, text: 'Girth (cm)' }
				}
			}
		}
	});
</script>

<ChartWrapper {config} isEmpty={data.length === 0} />
