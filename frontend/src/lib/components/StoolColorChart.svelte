<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart } from 'chart.js';
	import { COLOR_SWATCHES } from '$lib/stool-colors';

	interface StoolDataPoint {
		timestamp: string;
		color_score: number;
	}

	interface Props {
		data: StoolDataPoint[];
	}

	let { data }: Props = $props();
	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;

	function colorForScore(score: number): string {
		const swatch = COLOR_SWATCHES.find((s) => s.rating === score);
		return swatch?.color ?? '#999';
	}

	onMount(() => {
		const points = data.map((d) => ({
			x: new Date(d.timestamp).getTime(),
			y: d.color_score
		}));

		const pointColors = data.map((d) => colorForScore(d.color_score));

		chart = new Chart(canvas, {
			type: 'scatter',
			data: {
				datasets: [
					{
						label: 'Stool Color',
						data: points,
						pointBackgroundColor: pointColors,
						pointRadius: 6,
						pointBorderWidth: 1,
						pointBorderColor: '#333'
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
						min: 0.5,
						max: 7.5,
						title: { display: true, text: 'Color Score' },
						ticks: {
							stepSize: 1,
							callback: (value) => {
								const swatch = COLOR_SWATCHES.find((s) => s.rating === value);
								return swatch?.label ?? '';
							}
						}
					}
				},
				plugins: {
					legend: { display: false }
				}
			}
		});
	});

	onDestroy(() => {
		chart?.destroy();
	});
</script>

<canvas bind:this={canvas}></canvas>
