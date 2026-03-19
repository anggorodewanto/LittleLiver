<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart } from 'chart.js';

	interface AbdomenGirthDataPoint {
		timestamp: string;
		girth_cm: number;
	}

	interface Props {
		data: AbdomenGirthDataPoint[];
	}

	let { data }: Props = $props();
	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;
	let isEmpty = $derived(data.length === 0);

	onMount(() => {
		if (isEmpty) {
			return;
		}

		const points = data.map((d) => ({
			x: new Date(d.timestamp).getTime(),
			y: d.girth_cm
		}));

		chart = new Chart(canvas, {
			type: 'line',
			data: {
				datasets: [
					{
						label: 'Abdomen Girth',
						data: points,
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
							callback: (value) => new Date(value as number).toLocaleDateString()
						}
					},
					y: {
						title: { display: true, text: 'Girth (cm)' }
					}
				}
			}
		});
	});

	onDestroy(() => {
		chart?.destroy();
	});
</script>

{#if isEmpty}
	<p>No data</p>
{:else}
	<canvas bind:this={canvas}></canvas>
{/if}
