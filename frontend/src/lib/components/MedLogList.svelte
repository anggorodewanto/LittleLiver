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
			logs = data.med_logs;
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
