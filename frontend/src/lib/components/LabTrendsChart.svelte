<script lang="ts">
	import type { ChartConfiguration } from 'chart.js';
	import ChartWrapper from './ChartWrapper.svelte';
	import { dateTickCallback, LINE_COLORS } from '$lib/chart-utils';

	interface LabDataPoint {
		timestamp: string;
		test_name: string;
		value: string;
		unit: string;
	}

	interface Props {
		data: Record<string, LabDataPoint[]>;
		colors?: Map<string, string>;
	}

	let { data, colors }: Props = $props();

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

	let config = $derived.by<ChartConfiguration>(() => {
		const testNames = Object.keys(data);
		const datasets: Record<string, unknown>[] = [];
		let colorIdx = 0;

		for (const testName of testNames) {
			const points = data[testName];
			const color = colors?.get(testName) ?? LINE_COLORS[colorIdx % LINE_COLORS.length];
			const unit = points.length > 0 ? points[0].unit : '';

			const mappedPoints = points
				.map((p) => ({
					x: new Date(p.timestamp).getTime(),
					y: parseFloat(p.value)
				}))
				.filter((p) => !isNaN(p.y));

			datasets.push({
				label: unit ? `${testName} (${unit})` : testName,
				data: mappedPoints,
				borderColor: color,
				backgroundColor: color + '40',
				borderWidth: 2,
				pointRadius: 4,
				fill: false
			});

			// Add normal range shading if available, reusing mapped points for x-bounds
			const range = NORMAL_RANGES[testName];
			if (range && mappedPoints.length > 0) {
				const xValues = mappedPoints.map((p) => p.x);
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

		return {
			type: 'line' as const,
			data: { datasets },
			options: {
				responsive: true,
				plugins: {
					legend: { display: false }
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
						title: { display: true, text: 'Value' }
					}
				}
			}
		};
	});
</script>

<ChartWrapper {config} {isEmpty} />
