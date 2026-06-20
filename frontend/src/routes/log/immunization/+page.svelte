<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { activeBaby } from '$lib/stores/baby';
	import { apiClient } from '$lib/api';
	import { formatDateISO } from '$lib/datetime';
	import type {
		ScheduleEntry,
		ImmunizationReferenceResponse,
		ImmunizationRecord
	} from '$lib/types/immunization';

	const CUSTOM = '__custom__';

	let baby = $derived($activeBaby);
	let editId = $derived($page.url.searchParams.get('edit') ?? '');
	let paramCode = $derived($page.url.searchParams.get('code') ?? '');
	let paramName = $derived($page.url.searchParams.get('name') ?? '');
	let paramDose = $derived($page.url.searchParams.get('dose') ?? '');

	let reference = $state<ScheduleEntry[]>([]);

	// Selected option key in the picker: a reference key, CUSTOM, or '' (none).
	let selectedKey = $state('');
	let customName = $state('');
	let customDose = $state('');
	let administeredDate = $state(formatDateISO(new Date()));
	let provider = $state('');
	let lotNumber = $state('');
	let notes = $state('');

	let submitting = $state(false);
	let error = $state('');
	let validationError = $state('');
	let prefilled = false;

	function optionKey(entry: ScheduleEntry): string {
		return `${entry.code}::${entry.dose_number}`;
	}

	$effect(() => {
		apiClient
			.get<ImmunizationReferenceResponse>('/immunizations/reference')
			.then((data) => {
				reference = data.schedule ?? [];
			})
			.catch(() => {
				error = 'Failed to load vaccine list';
			});
	});

	// Pre-fill from query params (deep link from a schedule slot). Runs once.
	$effect(() => {
		if (prefilled || editId) return;
		if (!paramCode && !paramName) return;
		prefilled = true;
		if (paramCode) {
			selectedKey = `${paramCode}::${paramDose}`;
		} else {
			selectedKey = CUSTOM;
			customName = paramName;
			customDose = paramDose;
		}
	});

	// Edit mode: load the existing record and pre-fill all fields.
	$effect(() => {
		if (!editId || !baby) return;
		apiClient
			.get<ImmunizationRecord>(`/babies/${baby.id}/immunizations/${editId}`)
			.then((rec) => {
				administeredDate = rec.administered_date;
				provider = rec.provider ?? '';
				lotNumber = rec.lot_number ?? '';
				notes = rec.notes ?? '';
				const match = reference.find(
					(e) => e.code === rec.vaccine_code && e.dose_number === rec.dose_number
				);
				if (rec.vaccine_code && match) {
					selectedKey = optionKey(match);
				} else if (rec.vaccine_code) {
					selectedKey = `${rec.vaccine_code}::${rec.dose_number ?? ''}`;
				} else {
					selectedKey = CUSTOM;
					customName = rec.vaccine_name;
					customDose = rec.dose_number != null ? String(rec.dose_number) : '';
				}
			})
			.catch(() => {
				error = 'Failed to load entry';
			});
	});

	function resolveVaccine(): {
		vaccine_code?: string;
		vaccine_name: string;
		dose_number?: number;
	} | null {
		if (selectedKey === CUSTOM) {
			const name = customName.trim();
			if (!name) return null;
			const result: { vaccine_code?: string; vaccine_name: string; dose_number?: number } = {
				vaccine_name: name
			};
			if (customDose.trim()) result.dose_number = Number(customDose);
			return result;
		}
		const entry = reference.find((e) => optionKey(e) === selectedKey);
		if (!entry) return null;
		return {
			vaccine_code: entry.code,
			vaccine_name: entry.name,
			dose_number: entry.dose_number
		};
	}

	async function handleSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (!baby) return;

		const vaccine = resolveVaccine();
		if (!vaccine) {
			validationError = 'Please choose or enter a vaccine';
			return;
		}
		if (!administeredDate) {
			validationError = 'Date administered is required';
			return;
		}
		validationError = '';

		const body: Record<string, unknown> = {
			vaccine_name: vaccine.vaccine_name,
			administered_date: administeredDate
		};
		if (vaccine.vaccine_code) body.vaccine_code = vaccine.vaccine_code;
		if (vaccine.dose_number != null) body.dose_number = vaccine.dose_number;
		if (provider.trim()) body.provider = provider.trim();
		if (lotNumber.trim()) body.lot_number = lotNumber.trim();
		if (notes.trim()) body.notes = notes.trim();

		submitting = true;
		error = '';
		try {
			if (editId) {
				await apiClient.put(`/babies/${baby.id}/immunizations/${editId}`, body);
			} else {
				await apiClient.post(`/babies/${baby.id}/immunizations`, body);
			}
			goto('/immunizations');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			submitting = false;
		}
	}
</script>

<a href="/immunizations" class="back-link">&larr; Back</a>

{#if !baby}
	<p>No baby selected</p>
{:else}
	<h1>{editId ? 'Edit' : 'Log'} Immunization</h1>

	<form onsubmit={handleSubmit}>
		<div class="field">
			<label for="vax-picker">Vaccine</label>
			<select id="vax-picker" bind:value={selectedKey}>
				<option value="">Select a vaccine...</option>
				{#each reference as entry (optionKey(entry))}
					<option value={optionKey(entry)}>{entry.name} — {entry.dose_label}</option>
				{/each}
				<option value={CUSTOM}>Other (custom)</option>
			</select>
		</div>

		{#if selectedKey === CUSTOM}
			<div class="field">
				<label for="vax-name">Vaccine name</label>
				<input id="vax-name" type="text" bind:value={customName} />
			</div>
			<div class="field">
				<label for="vax-dose">Dose number (optional)</label>
				<input id="vax-dose" type="number" min="1" bind:value={customDose} />
			</div>
		{/if}

		<div class="field">
			<label for="vax-date">Date administered</label>
			<input id="vax-date" type="date" bind:value={administeredDate} />
		</div>

		<div class="field">
			<label for="vax-provider">Provider (optional)</label>
			<input id="vax-provider" type="text" bind:value={provider} />
		</div>

		<div class="field">
			<label for="vax-lot">Lot number (optional)</label>
			<input id="vax-lot" type="text" bind:value={lotNumber} />
		</div>

		<div class="field">
			<label for="vax-notes">Notes (optional)</label>
			<textarea id="vax-notes" bind:value={notes}></textarea>
		</div>

		{#if validationError}<p role="alert">{validationError}</p>{/if}
		{#if error}<p role="alert">{error}</p>{/if}

		<button type="submit" disabled={submitting}>
			{submitting ? 'Saving...' : editId ? 'Update Immunization' : 'Log Immunization'}
		</button>
	</form>
{/if}

<style>
	.back-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		margin-bottom: var(--space-3);
		min-height: var(--touch-target);
		text-decoration: none;
	}

	.back-link:hover {
		color: var(--color-primary);
	}

	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.field label {
		font-size: var(--font-size-sm);
		font-weight: 600;
	}

	textarea {
		min-height: 80px;
		resize: vertical;
	}

	button[type='submit'] {
		margin-top: var(--space-2);
		background: var(--color-primary);
		color: var(--color-text-inverse);
		min-height: 48px;
		font-weight: 600;
		border-radius: var(--radius-md);
	}

	button[type='submit']:hover {
		background: var(--color-primary-dark);
	}

	button[type='submit']:disabled {
		opacity: 0.6;
	}

	p[role='alert'] {
		color: var(--color-error);
		font-size: var(--font-size-sm);
		margin: 0;
	}
</style>
