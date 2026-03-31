<script lang="ts">
	import { untrack } from 'svelte';
	import { apiClient } from '$lib/api';
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';
	import type { Medication, MedicationsResponse } from '$lib/types/medication';

	export interface DoseLogPayload {
		medication_id: string;
		skipped: boolean;
		given_at?: string;
		skip_reason?: string;
		scheduled_time?: string;
		notes?: string;
	}

	export interface DoseLogInitialData {
		medication_id: string;
		skipped: boolean;
		given_at?: string;
		skip_reason?: string;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: DoseLogPayload) => void;
		babyId: string;
		medicationId?: string;
		scheduledTime?: string;
		initialData?: DoseLogInitialData;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, babyId, medicationId = '', scheduledTime, initialData, submitting = false, error = '' }: Props = $props();

	let medications = $state<Medication[]>([]);
	let selectedMedicationId = $state('');
	let status = $state<'given' | 'skipped' | ''>('');
	let givenAt = $state(defaultTimestamp());
	let skipReason = $state('');
	let notes = $state('');
	let validationError = $state('');
	let loadError = $state('');

	async function fetchMedications(): Promise<void> {
		loadError = '';
		try {
			const data = await apiClient.get<MedicationsResponse | Medication[]>(`/babies/${babyId}/medications`);
			const meds = Array.isArray(data) ? data : (data.medications ?? []);
			medications = meds.filter((m) => m.active);
		} catch {
			loadError = 'Failed to load medications';
		}
	}

	$effect(() => {
		void babyId;
		untrack(() => { void fetchMedications(); });
	});

	$effect(() => {
		selectedMedicationId = initialData?.medication_id ?? medicationId;
		status = initialData ? (initialData.skipped ? 'skipped' : 'given') : '';
		givenAt = initialData?.given_at ? fromISO8601(initialData.given_at) : defaultTimestamp();
		skipReason = initialData?.skip_reason ?? '';
		notes = initialData?.notes ?? '';
		validationError = '';
	});

	function selectStatus(s: 'given' | 'skipped'): void {
		status = s;
	}

	function handleSubmit(event: SubmitEvent): void {
		event.preventDefault();

		if (!selectedMedicationId) {
			validationError = 'Medication is required';
			return;
		}

		if (!status) {
			validationError = 'Select Given or Skipped';
			return;
		}

		validationError = '';
		const payload: DoseLogPayload = {
			medication_id: selectedMedicationId,
			skipped: status === 'skipped'
		};

		if (status === 'given') {
			payload.given_at = toISO8601(givenAt);
		}

		if (status === 'skipped' && skipReason.trim()) {
			payload.skip_reason = skipReason.trim();
		}

		if (scheduledTime) {
			if (/^\d{2}:\d{2}$/.test(scheduledTime)) {
				const [hours, minutes] = scheduledTime.split(':').map(Number);
				const d = new Date();
				d.setHours(hours, minutes, 0, 0);
				payload.scheduled_time = d.toISOString().replace(/\.\d{3}Z$/, 'Z');
			} else {
				payload.scheduled_time = scheduledTime;
			}
		}

		if (notes.trim()) {
			payload.notes = notes.trim();
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="dose-medication">Medication</label>
		<select id="dose-medication" bind:value={selectedMedicationId}>
			<option value="">Select medication...</option>
			{#each medications as med (med.id)}
				<option value={med.id}>{med.name} — {med.dose}</option>
			{/each}
		</select>
	</div>

	<div class="status-buttons">
		<button
			type="button"
			class="status-btn {status === 'given' ? 'selected' : ''}"
			onclick={() => selectStatus('given')}
		>Given</button>
		<button
			type="button"
			class="status-btn {status === 'skipped' ? 'selected' : ''}"
			onclick={() => selectStatus('skipped')}
		>Skipped</button>
	</div>

	{#if status === 'given'}
		<div>
			<label for="dose-given-at">Given At</label>
			<input id="dose-given-at" type="datetime-local" bind:value={givenAt} />
		</div>
	{/if}

	{#if status === 'skipped'}
		<div>
			<label for="dose-skip-reason">Skip Reason</label>
			<input id="dose-skip-reason" type="text" bind:value={skipReason} />
		</div>
	{/if}

	<div>
		<label for="dose-notes">Notes</label>
		<textarea id="dose-notes" bind:value={notes}></textarea>
	</div>

	{#if loadError}
		<p role="alert">{loadError}</p>
	{/if}

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : initialData ? 'Update Dose' : 'Log Dose'}
	</button>
</form>
