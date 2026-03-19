<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';
	import { dateTickCallback } from '$lib/chart-utils';

	interface TemperatureDataPoint {
		timestamp: string;
		value: number;
		method: string;
	}

	interface Props {
		data: TemperatureDataPoint[];
	}

	const FEVER_THRESHOLD = 38.0;

	let { data }: Props = $props();

	let config = $derived.by<ChartConfiguration>(() => {
		const tempPoints = data.map((d) => ({
			x: new Date(d.timestamp).getTime(),
			y: d.value
		}));

		const xValues = tempPoints.map((p) => p.x);
		const thresholdPoints =
			xValues.length > 0
				? [
						{ x: Math.min(...xValues), y: FEVER_THRESHOLD },
						{ x: Math.max(...xValues), y: FEVER_THRESHOLD }
					]
				: [];

		return {
			type: 'line' as const,
			data: {
				datasets: [
					{
						label: 'Temperature',
						data: tempPoints,
						borderColor: '#3b82f6',
						backgroundColor: '#3b82f680',
						borderWidth: 2,
						pointRadius: 4,
						fill: false
					},
					{
						label: 'Fever Threshold',
						data: thresholdPoints,
						borderColor: 'red',
						borderDash: [8, 4],
						borderWidth: 2,
						pointRadius: 0,
						fill: false
					}
				]
			},
			options: {
				responsive: true,
				scales: {
					x: {
						type: 'linear' as const,
						title: { display: true, text: 'Date' },
						ticks: {
							callback: dateTickCallback
						}
					},
					y: {
						title: { display: true, text: 'Temperature (°C)' }
					}
				}
			}
		};
	});
</script>

<ChartWrapper {config} isEmpty={data.length === 0} />
