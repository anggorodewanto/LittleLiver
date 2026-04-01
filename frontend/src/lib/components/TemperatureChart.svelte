<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';
	import { dateTickCallback, legendFilter } from '$lib/chart-utils';

	interface TemperatureDataPoint {
		timestamp: string;
		value: number;
		method: string;
	}

	interface Props {
		data: TemperatureDataPoint[];
	}

	const FEVER_THRESHOLDS: Record<string, number> = {
		rectal: 38.0,
		axillary: 37.5,
		ear: 38.0,
		forehead: 37.5
	};

	function getFeverThreshold(method: string): number {
		return FEVER_THRESHOLDS[method] ?? 38.0;
	}

	let { data }: Props = $props();

	let config = $derived.by<ChartConfiguration>(() => {
		const normalPoints = data
			.filter((d) => d.value < getFeverThreshold(d.method))
			.map((d) => ({
				x: new Date(d.timestamp).getTime(),
				y: d.value
			}));

		const feverPoints = data
			.filter((d) => d.value >= getFeverThreshold(d.method))
			.map((d) => ({
				x: new Date(d.timestamp).getTime(),
				y: d.value
			}));

		const xValues = data.map((d) => new Date(d.timestamp).getTime());
		// Show the lower threshold (37.5) so axillary/forehead fevers are visible
		const thresholdPoints =
			xValues.length > 0
				? [
						{ x: Math.min(...xValues), y: 37.5 },
						{ x: Math.max(...xValues), y: 37.5 }
					]
				: [];
		const upperThresholdPoints =
			xValues.length > 0
				? [
						{ x: Math.min(...xValues), y: 38.0 },
						{ x: Math.max(...xValues), y: 38.0 }
					]
				: [];

		return {
			type: 'line' as const,
			data: {
				datasets: [
					{
						label: 'Normal',
						data: normalPoints,
						borderColor: '#3b82f6',
						backgroundColor: '#3b82f680',
						borderWidth: 2,
						pointRadius: 4,
						fill: false,
						showLine: false
					},
					{
						label: 'Fever',
						data: feverPoints,
						borderColor: '#ef4444',
						backgroundColor: '#ef444480',
						borderWidth: 2,
						pointRadius: 6,
						fill: false,
						showLine: false
					},
					{
						label: 'Threshold (axillary/forehead)',
						data: thresholdPoints,
						borderColor: '#f59e0b',
						borderDash: [8, 4],
						borderWidth: 1,
						pointRadius: 0,
						fill: false
					},
					{
						label: 'Threshold (rectal/ear)',
						data: upperThresholdPoints,
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
				plugins: {
					legend: { labels: { filter: legendFilter } }
				},
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
