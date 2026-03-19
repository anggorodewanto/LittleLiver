<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart } from 'chart.js';

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
	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;

	onMount(() => {
		const tempPoints = data.map((d) => ({
			x: new Date(d.timestamp).getTime(),
			y: d.value
		}));

		// Fever threshold as horizontal line spanning same x range
		const xValues = tempPoints.map((p) => p.x);
		const thresholdPoints = xValues.length > 0
			? [{ x: Math.min(...xValues), y: FEVER_THRESHOLD }, { x: Math.max(...xValues), y: FEVER_THRESHOLD }]
			: [];

		chart = new Chart(canvas, {
			type: 'line',
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
						type: 'linear',
						title: { display: true, text: 'Date' },
						ticks: {
							callback: (value) => new Date(value as number).toLocaleDateString()
						}
					},
					y: {
						title: { display: true, text: 'Temperature (°C)' }
					}
				}
			}
		});
	});

	onDestroy(() => {
		chart?.destroy();
	});
</script>

<canvas bind:this={canvas}></canvas>
