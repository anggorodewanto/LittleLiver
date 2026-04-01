<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Chart, type ChartConfiguration } from 'chart.js';

	interface Props {
		config: ChartConfiguration;
		isEmpty?: boolean;
	}

	let { config, isEmpty = false }: Props = $props();
	let canvas = $state<HTMLCanvasElement>();
	let chart: Chart | null = null;

	$effect(() => {
		if (isEmpty || !canvas) {
			return;
		}

		// Access config to establish reactive dependency
		const currentConfig = config;

		if (chart) {
			chart.destroy();
			chart = null;
		}

		chart = new Chart(canvas, currentConfig);
	});

	onDestroy(() => {
		chart?.destroy();
	});
</script>

{#if isEmpty}
	<p>No data available</p>
{:else}
	<div style="position: relative; min-height: 250px;">
		<canvas bind:this={canvas}></canvas>
	</div>
{/if}
