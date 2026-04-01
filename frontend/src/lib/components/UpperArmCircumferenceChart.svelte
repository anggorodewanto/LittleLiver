<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';
	import { dateTickCallback } from '$lib/chart-utils';

	interface UpperArmCircumferenceDataPoint {
		timestamp: string;
		circumference_cm: number;
	}

	interface Props {
		data: UpperArmCircumferenceDataPoint[];
	}

	let { data }: Props = $props();

	let config = $derived<ChartConfiguration>({
		type: 'line',
		data: {
			datasets: [
				{
					label: 'Upper Arm Circumference',
					data: data.map((d) => ({
						x: new Date(d.timestamp).getTime(),
						y: d.circumference_cm
					})),
					borderColor: '#f59e0b',
					backgroundColor: '#f59e0b80',
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
					title: { display: true, text: 'Upper Arm Circumference (cm)' }
				}
			}
		}
	});
</script>

<ChartWrapper {config} isEmpty={data.length === 0} />
