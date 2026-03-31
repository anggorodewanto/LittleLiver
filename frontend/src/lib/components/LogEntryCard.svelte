<script lang="ts">
	import { goto } from '$app/navigation';
	import { formatDateTime } from '$lib/datetime';
	import type { LogTypeConfig } from '$lib/types/logs';

	interface Props {
		entry: Record<string, unknown>;
		logType: LogTypeConfig;
		ondelete: (id: string) => void;
		medNames?: Record<string, string>;
	}

	let { entry, logType, ondelete, medNames = {} }: Props = $props();

	let confirmingDelete = $state(false);

	function handleEdit(): void {
		goto(`/log/${logType.metricParam}?edit=${entry.id}`);
	}

	function handleDeleteClick(): void {
		confirmingDelete = true;
	}

	function handleConfirmDelete(): void {
		confirmingDelete = false;
		ondelete(entry.id as string);
	}

	function handleCancelDelete(): void {
		confirmingDelete = false;
	}

	function capitalize(str: unknown): string {
		if (!str || typeof str !== 'string') return '';
		return str.charAt(0).toUpperCase() + str.slice(1);
	}

	function truncate(str: unknown, max: number): string {
		if (!str || typeof str !== 'string') return '';
		if (str.length <= max) return str;
		return str.slice(0, max) + '...';
	}

	function entryTimestamp(): string {
		if (entry.timestamp) return entry.timestamp as string;
		// Med-logs don't have timestamp — use given_at or created_at
		if (entry.given_at) return entry.given_at as string;
		if (entry.created_at) return entry.created_at as string;
		return '';
	}
</script>

<div class="log-entry-card">
	<div class="card-header">
		<span class="timestamp">{formatDateTime(entryTimestamp())}</span>
	</div>

	<div class="card-body">
		{#if logType.key === 'feeding'}
			{#if entry.feed_type}<div class="field">{capitalize(entry.feed_type)}</div>{/if}
			{#if entry.volume_ml}<div class="field">{entry.volume_ml} mL</div>{/if}
			{#if entry.calories}<div class="field">{entry.calories} kcal</div>{/if}
			{#if entry.duration_min}<div class="field">{entry.duration_min} min</div>{/if}
		{:else if logType.key === 'stool'}
			{#if entry.color_rating}<div class="field">Color: {entry.color_rating}/7</div>{/if}
			{#if entry.consistency}<div class="field">{capitalize(entry.consistency)}</div>{/if}
			{#if entry.volume_estimate}<div class="field">{capitalize(entry.volume_estimate)}</div>{/if}
		{:else if logType.key === 'urine'}
			{#if entry.color}<div class="field">{capitalize(entry.color)}</div>{/if}
			{#if entry.volume_ml}<div class="field">{entry.volume_ml} mL</div>{/if}
		{:else if logType.key === 'weight'}
			{#if entry.weight_kg}<div class="field">{entry.weight_kg} kg</div>{/if}
			{#if entry.measurement_source}<div class="field">{capitalize(entry.measurement_source)}</div>{/if}
		{:else if logType.key === 'temperature'}
			{#if entry.value}<div class="field">{entry.value} °C</div>{/if}
			{#if entry.method}<div class="field">{capitalize(entry.method)}</div>{/if}
		{:else if logType.key === 'abdomen'}
			{#if entry.firmness}<div class="field">{capitalize(entry.firmness)}</div>{/if}
			{#if entry.tenderness != null}<div class="field">Tenderness: {entry.tenderness ? 'Yes' : 'No'}</div>{/if}
			{#if entry.girth_cm}<div class="field">{entry.girth_cm} cm</div>{/if}
		{:else if logType.key === 'skin'}
			{#if entry.jaundice_level}<div class="field">Jaundice: {entry.jaundice_level}</div>{/if}
			{#if entry.scleral_icterus != null}<div class="field">Scleral icterus: {entry.scleral_icterus ? 'Yes' : 'No'}</div>{/if}
		{:else if logType.key === 'bruising'}
			{#if entry.location}<div class="field">{capitalize(entry.location)}</div>{/if}
			{#if entry.size_estimate}<div class="field">{capitalize(entry.size_estimate)}</div>{/if}
			{#if entry.color}<div class="field">{capitalize(entry.color)}</div>{/if}
		{:else if logType.key === 'lab'}
			{#if entry.test_name}<div class="field">{entry.test_name}</div>{/if}
			{#if entry.value}<div class="field">{entry.value} {entry.unit ?? ''}</div>{/if}
		{:else if logType.key === 'note'}
			{#if entry.category}<div class="field">{capitalize(entry.category)}</div>{/if}
			{#if entry.content}<div class="field">{truncate(entry.content, 100)}</div>{/if}
		{:else if logType.key === 'med-log'}
			{@const medName = medNames[entry.medication_id as string]}
			{#if medName}<div class="field"><strong>{medName}</strong></div>{/if}
			{#if entry.skipped}
				<div class="field"><span class="badge-skipped">Skipped</span></div>
			{:else}
				<div class="field"><span class="badge-given">Given</span></div>
			{/if}
			{#if entry.skip_reason}<div class="field">{entry.skip_reason}</div>{/if}
		{:else if logType.key === 'fluid'}
			{#if entry.direction}<div class="field">{entry.direction === 'intake' ? 'Intake' : 'Output'}</div>{/if}
			{#if entry.method}<div class="field">{capitalize(entry.method)}</div>{/if}
			{#if entry.volume_ml}<div class="field">{entry.volume_ml} mL</div>{/if}
		{/if}

		{#if entry.notes && logType.key !== 'note'}
			<div class="notes">{truncate(entry.notes, 100)}</div>
		{/if}
	</div>

	<div class="card-footer">
		{#if confirmingDelete}
			<span class="confirm-text">Are you sure?</span>
			<button class="btn-confirm" onclick={handleConfirmDelete}>Confirm</button>
			<button class="btn-cancel" onclick={handleCancelDelete}>Cancel</button>
		{:else}
			<button class="btn-edit" onclick={handleEdit}>Edit</button>
			<button class="btn-delete" onclick={handleDeleteClick}>Delete</button>
		{/if}
	</div>
</div>

<style>
	.log-entry-card {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-3) var(--space-4);
		box-shadow: var(--shadow-sm);
	}

	.card-header {
		margin-bottom: var(--space-2);
	}

	.timestamp {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	.card-body {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		margin-bottom: var(--space-2);
	}

	.field {
		font-size: var(--font-size-sm);
		color: var(--color-text);
	}

	.notes {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		margin-top: var(--space-1);
	}

	.badge-skipped {
		display: inline-block;
		padding: 2px var(--space-2);
		border-radius: var(--radius-md);
		font-size: var(--font-size-xs);
		font-weight: 600;
		background: var(--color-warning-bg, #fef3c7);
		color: var(--color-warning, #d97706);
	}

	.badge-given {
		display: inline-block;
		padding: 2px var(--space-2);
		border-radius: var(--radius-md);
		font-size: var(--font-size-xs);
		font-weight: 600;
		background: var(--color-success-bg, #dcfce7);
		color: var(--color-success, #16a34a);
	}

	.card-footer {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		border-top: 1px solid var(--color-border);
		padding-top: var(--space-2);
	}

	.confirm-text {
		font-size: var(--font-size-sm);
		color: var(--color-text);
	}

	.btn-edit,
	.btn-delete,
	.btn-confirm,
	.btn-cancel {
		min-height: var(--touch-target);
		padding: var(--space-1) var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		font-size: var(--font-size-sm);
		cursor: pointer;
		background: var(--color-surface);
		color: var(--color-text);
	}

	.btn-edit {
		color: var(--color-primary);
		border-color: var(--color-primary);
	}

	.btn-delete {
		color: var(--color-error);
		border-color: var(--color-error);
	}

	.btn-confirm {
		color: var(--color-error);
		border-color: var(--color-error);
	}

	.btn-cancel {
		color: var(--color-text-muted);
	}
</style>
