<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart } from 'chart.js';

	interface DiaperDailyDataPoint {
		date: string;
		wet_count: number;
		stool_count: number;
	}

	interface Props {
		data: DiaperDailyDataPoint[];
	}

	let { data }: Props = $props();
	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;
	let isEmpty = $derived(data.length === 0);

	onMount(() => {
		if (isEmpty) {
			return;
		}

		const labels = data.map((d) => d.date);

		chart = new Chart(canvas, {
			type: 'bar',
			data: {
				labels,
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
