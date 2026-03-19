<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart } from 'chart.js';

	interface LabDataPoint {
		timestamp: string;
		test_name: string;
		value: string;
		unit: string;
	}

	interface Props {
		data: Record<string, LabDataPoint[]>;
	}

	let { data }: Props = $props();
	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;

	const LINE_COLORS = ['#ef4444', '#3b82f6', '#22c55e', '#f59e0b', '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16'];

	// Normal ranges for common Kasai-relevant labs
	const NORMAL_RANGES: Record<string, { min: number; max: number }> = {
		total_bilirubin: { min: 0, max: 2.0 },
		direct_bilirubin: { min: 0, max: 0.3 },
		ALT: { min: 0, max: 40 },
		AST: { min: 0, max: 40 },
		GGT: { min: 0, max: 50 },
		albumin: { min: 3.5, max: 5.0 },
		INR: { min: 0.8, max: 1.2 },
		platelets: { min: 150, max: 400 }
	};

	let isEmpty = $derived(Object.keys(data).length === 0);

	onMount(() => {
		if (isEmpty) {
			return;
		}

		const testNames = Object.keys(data);

		const datasets: Record<string, unknown>[] = [];
		let colorIdx = 0;

		for (const testName of testNames) {
			const points = data[testName];
			const color = LINE_COLORS[colorIdx % LINE_COLORS.length];
			const unit = points.length > 0 ? points[0].unit : '';

			datasets.push({
				label: unit ? `${testName} (${unit})` : testName,
				data: points
					.map((p) => ({
						x: new Date(p.timestamp).getTime(),
						y: parseFloat(p.value)
					}))
					.filter((p) => !isNaN(p.y)),
				borderColor: color,
				backgroundColor: color + '40',
				borderWidth: 2,
				pointRadius: 4,
				fill: false
			});

			// Add normal range shading if available
			const range = NORMAL_RANGES[testName];
			if (range && points.length > 0) {
				const xValues = points.map((p) => new Date(p.timestamp).getTime());
				const xMin = Math.min(...xValues);
				const xMax = Math.max(...xValues);

				datasets.push({
					label: `Normal (${testName})`,
					data: [
						{ x: xMin, y: range.max },
						{ x: xMax, y: range.max }
					],
					borderColor: color + '30',
					backgroundColor: color + '15',
					borderWidth: 0,
					pointRadius: 0,
					fill: {
						target: { value: range.min },
						above: color + '15'
					}
				});
			}

			colorIdx++;
		}

		chart = new Chart(canvas, {
			type: 'line',
			data: { datasets },
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
						title: { display: true, text: 'Value' }
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
