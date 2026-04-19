<script lang="ts">
	import { goto } from '$app/navigation';
	import { formatTime } from '$lib/datetime';
	import type { LogTypeConfig } from '$lib/types/logs';
	import PhotoLightbox from './PhotoLightbox.svelte';

	interface Photo {
		key: string;
		url: string;
		thumbnail_url: string;
	}

	interface Props {
		entry: Record<string, unknown>;
		logType: LogTypeConfig;
		ondelete: (id: string) => void;
		medNames?: Record<string, string>;
	}

	let { entry, logType, ondelete, medNames = {} }: Props = $props();

	let confirmingDelete = $state(false);
	let lightboxOpen = $state(false);

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
		if (entry.given_at) return entry.given_at as string;
		if (entry.created_at) return entry.created_at as string;
		return '';
	}

	function summaryParts(): string[] {
		const parts: string[] = [];

		if (logType.key === 'feeding') {
			if (entry.feed_type) parts.push(capitalize(entry.feed_type));
			if (entry.volume_ml) parts.push(`${entry.volume_ml} mL`);
			if (entry.calories) parts.push(`${entry.calories} kcal`);
			if (entry.duration_min) parts.push(`${entry.duration_min} min`);
		} else if (logType.key === 'stool') {
			if (entry.color_rating) parts.push(`Color ${entry.color_rating}/7`);
			if (entry.consistency) parts.push(capitalize(entry.consistency));
			if (entry.volume_estimate) parts.push(capitalize(entry.volume_estimate));
		} else if (logType.key === 'urine') {
			if (entry.color) parts.push(capitalize(entry.color));
			if (entry.volume_ml) parts.push(`${entry.volume_ml} mL`);
		} else if (logType.key === 'weight') {
			if (entry.weight_kg) parts.push(`${entry.weight_kg} kg`);
			if (entry.measurement_source) parts.push(capitalize(entry.measurement_source));
		} else if (logType.key === 'temperature') {
			if (entry.value) parts.push(`${entry.value} °C`);
			if (entry.method) parts.push(capitalize(entry.method));
		} else if (logType.key === 'abdomen') {
			if (entry.firmness) parts.push(capitalize(entry.firmness));
			if (entry.tenderness != null) parts.push(entry.tenderness ? 'Tender' : 'Not tender');
			if (entry.girth_cm) parts.push(`${entry.girth_cm} cm`);
		} else if (logType.key === 'skin') {
			if (entry.jaundice_level) parts.push(`Jaundice: ${entry.jaundice_level}`);
			if (entry.scleral_icterus != null) parts.push(`Scleral: ${entry.scleral_icterus ? 'Yes' : 'No'}`);
		} else if (logType.key === 'bruising') {
			if (entry.location) parts.push(capitalize(entry.location));
			if (entry.size_estimate) parts.push(capitalize(entry.size_estimate));
			if (entry.color) parts.push(capitalize(entry.color));
		} else if (logType.key === 'lab') {
			if (entry.test_name) parts.push(entry.test_name as string);
			if (entry.value) parts.push(`${entry.value}${entry.unit ? ' ' + entry.unit : ''}`);
		} else if (logType.key === 'note') {
			if (entry.category) parts.push(capitalize(entry.category));
			if (entry.content) parts.push(truncate(entry.content, 60));
		} else if (logType.key === 'med-log') {
			const medName = medNames[entry.medication_id as string];
			if (medName) parts.push(medName);
			if (entry.skipped) {
				parts.push('Skipped');
				if (entry.skip_reason) parts.push(entry.skip_reason as string);
			} else {
				parts.push('Given');
			}
		} else if (logType.key === 'fluid') {
			if (entry.direction) parts.push(entry.direction === 'intake' ? 'Intake' : 'Output');
			if (entry.method) parts.push(capitalize(entry.method));
			if (entry.volume_ml) parts.push(`${entry.volume_ml} mL`);
		} else if (logType.key === 'head-circumference') {
			if (entry.circumference_cm) parts.push(`${entry.circumference_cm} cm`);
			if (entry.measurement_source) parts.push(capitalize(entry.measurement_source));
		} else if (logType.key === 'upper-arm-circumference') {
			if (entry.circumference_cm) parts.push(`${entry.circumference_cm} cm`);
			if (entry.measurement_source) parts.push(capitalize(entry.measurement_source));
		}

		return parts;
	}
</script>

<div class="log-entry-row">
	<span class="row-time">{formatTime(entryTimestamp())}</span>

	{#if confirmingDelete}
		<span class="row-confirm">Delete?</span>
		<button class="btn-sm btn-confirm" onclick={handleConfirmDelete}>Confirm</button>
		<button class="btn-sm btn-cancel" onclick={handleCancelDelete}>Cancel</button>
	{:else}
		<span class="row-summary">
			{#each summaryParts() as part, i}
				{#if i > 0}<span class="sep"> · </span>{/if}{part}
			{/each}
		</span>
		{#if Array.isArray(entry.photos) && (entry.photos as Photo[]).length > 0}
			<button
				type="button"
				class="photo-indicator"
				aria-label="{(entry.photos as Photo[]).length} photo(s)"
				onclick={() => { lightboxOpen = true; }}
			>📷 {(entry.photos as Photo[]).length}</button>
		{/if}
		<div class="row-actions">
			<button class="btn-sm btn-edit" onclick={handleEdit} aria-label="Edit">✏️</button>
			<button class="btn-sm btn-delete" onclick={handleDeleteClick} aria-label="Delete">🗑️</button>
		</div>
	{/if}

	{#if lightboxOpen && Array.isArray(entry.photos) && (entry.photos as Photo[]).length > 0}
		<PhotoLightbox photos={entry.photos as Photo[]} onclose={() => { lightboxOpen = false; }} />
	{/if}
</div>

<style>
	.log-entry-row {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--color-border);
		min-height: var(--touch-target);
	}

	.row-time {
		flex-shrink: 0;
		width: 4.5em;
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	.row-summary {
		flex: 1;
		font-size: var(--font-size-sm);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.sep {
		color: var(--color-text-muted);
	}

	.photo-indicator {
		flex-shrink: 0;
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		background: none;
		border: none;
		padding: var(--space-1) var(--space-2);
		cursor: pointer;
		min-height: var(--touch-target);
	}

	.photo-indicator:hover {
		color: var(--color-text);
	}

	.row-confirm {
		flex: 1;
		font-size: var(--font-size-sm);
		color: var(--color-text);
	}

	.row-actions {
		flex-shrink: 0;
		display: flex;
		gap: var(--space-1);
	}

	.btn-sm {
		min-height: var(--touch-target);
		padding: var(--space-1) var(--space-2);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		font-size: var(--font-size-xs);
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
