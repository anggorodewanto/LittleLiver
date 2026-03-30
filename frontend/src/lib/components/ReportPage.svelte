<script lang="ts">
	import { apiClient } from '$lib/api';

	interface Props {
		babyId: string;
		babyName: string;
	}

	let { babyId, babyName }: Props = $props();

	let fromDate = $state('');
	let toDate = $state('');
	let loading = $state(false);
	let error = $state<string | null>(null);

	let datesValid = $derived(
		fromDate !== '' && toDate !== '' && fromDate <= toDate
	);

	function formatDisplayDate(dateStr: string): string {
		const date = new Date(dateStr + 'T00:00:00');
		return date.toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'long',
			day: 'numeric'
		});
	}

	async function generateReport(): Promise<void> {
		if (!datesValid) {
			return;
		}

		loading = true;
		error = null;

		try {
			const response = await apiClient.getRaw(
				`/babies/${babyId}/report?from=${fromDate}&to=${toDate}`
			);

			const blob = await response.blob();
			const url = URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = url;
			link.download = `report-${babyName.toLowerCase()}-${fromDate}-to-${toDate}.pdf`;
			link.click();
			setTimeout(() => URL.revokeObjectURL(url), 10000);
		} catch (e) {
			const statusMatch = e instanceof Error && e.message.match(/(\d{3})/);
			const status = statusMatch ? statusMatch[1] : 'unknown';
			error = `Failed to generate report (${status})`;
		} finally {
			loading = false;
		}
	}
</script>

<div class="report-page">
	<h2>Clinical Report</h2>

	<div class="date-picker">
		<label>
			From
			<input type="date" bind:value={fromDate} />
		</label>
		<label>
			To
			<input type="date" bind:value={toDate} />
		</label>
	</div>

	{#if datesValid}
		<div class="preview-summary">
			<h3>Report for {babyName}</h3>
			<p>{formatDisplayDate(fromDate)} to {formatDisplayDate(toDate)}</p>
			<p>The report will include:</p>
			<ul>
				<li>Stool color log and trend chart</li>
				<li>Weight chart with WHO percentiles</li>
				<li>Lab trends</li>
				<li>Temperature log</li>
				<li>Feeding summary</li>
				<li>Medication adherence</li>
				<li>Notable observations and photos</li>
			</ul>
		</div>
	{/if}

	{#if error}
		<div class="error">{error}</div>
	{/if}

	<button
		type="button"
		disabled={!datesValid || loading}
		onclick={generateReport}
	>
		{#if loading}
			Generating...
		{:else}
			Generate Report
		{/if}
	</button>
</div>

<style>
	.date-picker {
		display: flex;
		gap: var(--space-3);
		margin-bottom: var(--space-4);
	}

	.date-picker label {
		flex: 1;
	}

	.preview-summary {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
		box-shadow: var(--shadow-sm);
	}

	.preview-summary ul {
		padding-left: var(--space-5);
	}

	.preview-summary li {
		margin-bottom: var(--space-1);
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	.report-page button {
		width: 100%;
		background: var(--color-primary);
		color: var(--color-text-inverse);
		min-height: 48px;
		font-weight: 600;
		border-radius: var(--radius-md);
	}

	.report-page button:hover {
		background: var(--color-primary-dark);
	}
</style>
