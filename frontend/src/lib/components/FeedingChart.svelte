<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart } from 'chart.js';

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
	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;
	let isEmpty = $derived(data.length === 0);

	onMount(() => {
		if (isEmpty) {
			return;
		}

		const labels = data.map((d) => d.date);
		const calories = data.map((d) => d.total_calories);

		chart = new Chart(canvas, {
			type: 'bar',
			data: {
				labels,
				datasets: [
					{
						label: 'Daily Calories',
						data: calories,
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
