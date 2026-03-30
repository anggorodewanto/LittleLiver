<script lang="ts">
	import { untrack } from 'svelte';
	import { apiClient } from '$lib/api';
	import { formatDateTime } from '$lib/datetime';

	interface MedLog {
		id: string;
		medication_id: string;
		baby_id: string;
		scheduled_time: string | null;
		given_at: string | null;
		skipped: boolean;
		skip_reason: string | null;
		notes: string | null;
		created_at: string;
	}

	interface MedLogsResponse {
		med_logs: MedLog[];
	}

	interface Props {
		babyId: string;
		medicationId: string;
	}

	let { babyId, medicationId }: Props = $props();

	let loading = $state(true);
	let error = $state<string | null>(null);
	let logs = $state<MedLog[]>([]);

	async function fetchLogs(): Promise<void> {
		loading = true;
		error = null;
		try {
			const data = await apiClient.get<MedLogsResponse>(
				`/babies/${babyId}/med-logs?medication_id=${medicationId}`
			);
			logs = [...data.med_logs].sort((a, b) => {
				const ta = a.scheduled_time ?? a.created_at;
				const tb = b.scheduled_time ?? b.created_at;
				return tb.localeCompare(ta);
			});
		} catch {
			error = 'Failed to load dose logs';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void babyId;
		void medicationId;
		untrack(() => { void fetchLogs(); });
	});
</script>

{#if loading}
	<div class="loading">Loading...</div>
{:else if error}
	<div class="error">{error}</div>
{:else if logs.length === 0}
	<div class="empty">No dose logs found.</div>
{:else}
	<div class="med-log-list">
		{#each logs as log (log.id)}
			<div class="med-log-item" data-testid="med-log-item">
				<div class="log-status">
					{#if log.skipped}
						<span class="status-skipped">Skipped</span>
						{#if log.skip_reason}
							<span class="skip-reason">{log.skip_reason}</span>
						{/if}
					{:else}
						<span class="status-given">Given</span>
						{#if log.given_at}
							<span class="given-time">{formatDateTime(log.given_at)}</span>
						{/if}
					{/if}
				</div>
				{#if log.notes}
					<div class="log-notes">{log.notes}</div>
				{/if}
				{#if log.scheduled_time}
					<div class="log-scheduled">Scheduled: {formatDateTime(log.scheduled_time)}</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}

<style>
	.med-log-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.med-log-item {
		padding: var(--space-3) var(--space-4);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-sm);
	}

	.log-status {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-wrap: wrap;
	}

	.status-given {
		display: inline-block;
		padding: 2px var(--space-2);
		border-radius: var(--radius-full);
		font-size: var(--font-size-xs);
		font-weight: 600;
		background: var(--color-success-bg);
		color: var(--color-success);
	}

	.status-skipped {
		display: inline-block;
		padding: 2px var(--space-2);
		border-radius: var(--radius-full);
		font-size: var(--font-size-xs);
		font-weight: 600;
		background: var(--color-warning-bg);
		color: var(--color-warning);
	}

	.skip-reason,
	.given-time {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	.log-notes {
		font-size: var(--font-size-sm);
		color: var(--color-text);
		margin-top: var(--space-2);
	}

	.log-scheduled {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		margin-top: var(--space-1);
	}
</style>
