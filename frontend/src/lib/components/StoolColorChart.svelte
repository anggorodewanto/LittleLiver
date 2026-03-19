<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';
	import { COLOR_SWATCHES } from '$lib/stool-colors';
	import { dateTickCallback } from '$lib/chart-utils';

	interface StoolDataPoint {
		timestamp: string;
		color_score: number;
	}

	interface Props {
		data: StoolDataPoint[];
	}

	let { data }: Props = $props();

	function colorForScore(score: number): string {
		const swatch = COLOR_SWATCHES.find((s) => s.rating === score);
		return swatch?.color ?? '#999';
	}

	let config = $derived<ChartConfiguration>({
		type: 'scatter',
		data: {
			datasets: [
				{
					label: 'Stool Color',
					data: data.map((d) => ({
						x: new Date(d.timestamp).getTime(),
						y: d.color_score
					})),
					pointBackgroundColor: data.map((d) => colorForScore(d.color_score)),
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
						callback: dateTickCallback
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
</script>

<ChartWrapper {config} isEmpty={data.length === 0} />
