<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart } from 'chart.js';

	interface WeightDataPoint {
		timestamp: string;
		weight_kg: number;
		measurement_source: string;
	}

	interface PercentilePoint {
		age_days: number;
		weight_kg: number;
	}

	interface Percentiles {
		p3: PercentilePoint[];
		p15: PercentilePoint[];
		p50: PercentilePoint[];
		p85: PercentilePoint[];
		p97: PercentilePoint[];
	}

	interface Props {
		data: WeightDataPoint[];
		percentiles: Percentiles;
		dateOfBirth: string;
	}

	let { data, percentiles, dateOfBirth }: Props = $props();
	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;

	function ageDaysForTimestamp(timestamp: string): number {
		const birth = new Date(dateOfBirth).getTime();
		const ts = new Date(timestamp).getTime();
		return Math.floor((ts - birth) / (24 * 60 * 60 * 1000));
	}

	const PERCENTILE_COLORS: Record<string, string> = {
		'3rd': '#ef444480',
		'15th': '#f97316a0',
		'50th': '#22c55ea0',
		'85th': '#f97316a0',
		'97th': '#ef444480'
	};

	onMount(() => {
		const weightPoints = data.map((d) => ({
			x: ageDaysForTimestamp(d.timestamp),
			y: d.weight_kg
		}));

		const percentileEntries: [string, string, PercentilePoint[]][] = [
			['3rd', 'p3', percentiles.p3],
			['15th', 'p15', percentiles.p15],
			['50th', 'p50', percentiles.p50],
			['85th', 'p85', percentiles.p85],
			['97th', 'p97', percentiles.p97]
		];

		const percentileDatasets = percentileEntries.map(([label, , points]) => ({
			label,
			data: points.map((p) => ({ x: p.age_days, y: p.weight_kg })),
			borderColor: PERCENTILE_COLORS[label],
			borderDash: [5, 5],
			borderWidth: 1,
			pointRadius: 0,
			fill: false
		}));

		chart = new Chart(canvas, {
			type: 'line',
			data: {
				datasets: [
					{
						label: 'Weight',
						data: weightPoints,
						borderColor: '#3b82f6',
						backgroundColor: '#3b82f680',
						borderWidth: 2,
						pointRadius: 4,
						fill: false
					},
					...percentileDatasets
				]
			},
			options: {
				responsive: true,
				scales: {
					x: {
						type: 'linear',
						title: { display: true, text: 'Age (days)' }
					},
					y: {
						title: { display: true, text: 'Weight (kg)' }
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
