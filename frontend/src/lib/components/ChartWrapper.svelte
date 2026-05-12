<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Chart, type ChartConfiguration } from 'chart.js';

	interface Props {
		config: ChartConfiguration;
		isEmpty?: boolean;
	}

	let { config, isEmpty = false }: Props = $props();
	let canvas = $state<HTMLCanvasElement>();
	let fullCanvas = $state<HTMLCanvasElement>();
	let chart: Chart | null = null;
	let fullChart: Chart | null = null;
	let expanded = $state(false);

	$effect(() => {
		if (isEmpty || !canvas) return;
		const currentConfig = config;
		if (chart) {
			chart.destroy();
			chart = null;
		}
		chart = new Chart(canvas, currentConfig);
	});

	function buildZoomConfig(base: ChartConfiguration): ChartConfiguration {
		const cloned: ChartConfiguration = {
			...base,
			options: {
				...(base.options ?? {}),
				maintainAspectRatio: false,
				plugins: {
					...((base.options as { plugins?: Record<string, unknown> })?.plugins ?? {}),
					zoom: {
						pan: { enabled: true, mode: 'xy' },
						zoom: {
							wheel: { enabled: true },
							pinch: { enabled: true },
							mode: 'xy'
						}
					}
				}
			}
		};
		return cloned;
	}

	$effect(() => {
		if (!expanded || !fullCanvas) return;
		if (fullChart) {
			fullChart.destroy();
			fullChart = null;
		}
		fullChart = new Chart(fullCanvas, buildZoomConfig(config));
	});

	function open(): void {
		expanded = true;
	}

	function close(): void {
		expanded = false;
		if (fullChart) {
			fullChart.destroy();
			fullChart = null;
		}
	}

	function resetZoom(): void {
		(fullChart as unknown as { resetZoom?: () => void })?.resetZoom?.();
	}

	function onKeydown(e: KeyboardEvent): void {
		if (e.key === 'Escape') close();
	}

	onDestroy(() => {
		chart?.destroy();
		fullChart?.destroy();
	});
</script>

<svelte:window onkeydown={onKeydown} />

{#if isEmpty}
	<p>No data available</p>
{:else}
	<button
		type="button"
		class="chart-trigger"
		data-testid="chart-expand"
		onclick={open}
		aria-label="Expand chart"
	>
		<canvas bind:this={canvas}></canvas>
	</button>
{/if}

{#if expanded}
	<div
		class="chart-modal-overlay"
		data-testid="chart-modal"
		role="dialog"
		aria-modal="true"
		aria-label="Expanded chart"
	>
		<div class="chart-modal">
			<div class="chart-modal-toolbar">
				<button type="button" class="toolbar-btn" onclick={resetZoom}>Reset zoom</button>
				<button
					type="button"
					class="toolbar-btn close-btn"
					data-testid="chart-modal-close"
					onclick={close}
					aria-label="Close"
				>
					✕
				</button>
			</div>
			<div class="chart-modal-body">
				<canvas bind:this={fullCanvas}></canvas>
			</div>
			<p class="chart-modal-hint">Wheel/pinch to zoom · drag to pan · Esc to close</p>
		</div>
	</div>
{/if}

<style>
	.chart-trigger {
		display: block;
		position: relative;
		min-height: 250px;
		width: 100%;
		padding: 0;
		margin: 0;
		background: transparent;
		border: none;
		cursor: zoom-in;
	}

	.chart-trigger canvas {
		width: 100% !important;
	}

	.chart-modal-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.72);
		z-index: 1000;
		display: flex;
		align-items: stretch;
		justify-content: stretch;
		padding: var(--space-3, 12px);
	}

	.chart-modal {
		background: var(--color-surface, #fff);
		border-radius: var(--radius-md, 8px);
		flex: 1;
		display: flex;
		flex-direction: column;
		padding: var(--space-3, 12px);
		gap: var(--space-2, 8px);
		min-height: 0;
	}

	.chart-modal-toolbar {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2, 8px);
	}

	.toolbar-btn {
		background: var(--color-bg, #f4f4f5);
		border: 1px solid var(--color-border, #ddd);
		border-radius: var(--radius-sm, 4px);
		padding: 6px 12px;
		cursor: pointer;
		font-size: 14px;
	}

	.toolbar-btn:hover {
		background: var(--color-border, #e5e5e5);
	}

	.close-btn {
		font-weight: bold;
	}

	.chart-modal-body {
		flex: 1;
		min-height: 0;
		position: relative;
	}

	.chart-modal-body canvas {
		width: 100% !important;
		height: 100% !important;
	}

	.chart-modal-hint {
		margin: 0;
		text-align: center;
		font-size: 12px;
		color: var(--color-text-muted, #666);
	}
</style>
