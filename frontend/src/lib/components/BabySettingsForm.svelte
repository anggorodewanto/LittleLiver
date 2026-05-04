<script lang="ts">
	import type { Baby, UpdateBabyInput } from '$lib/stores/baby';

	interface Props {
		baby: Baby;
		onsave: (data: UpdateBabyInput, recalculate: boolean) => void;
		submitting?: boolean;
		error?: string;
	}

	let { baby, onsave, submitting = false, error = '' }: Props = $props();

	let name = $state('');
	let dateOfBirth = $state('');
	let sex: 'male' | 'female' = $state('male');
	let diagnosisDate = $state('');
	let kasaiDate = $state('');
	let defaultCalPerFeed = $state('67');
	let notes = $state('');
	let gestationalWeeks = $state('');
	let gestationalDays = $state('');
	let recalculate = $state(false);
	let validationError = $state('');

	// Sync form state from baby prop (handles initial load and baby switching)
	let syncedBabyId = $state('');
	$effect(() => {
		if (baby.id !== syncedBabyId) {
			syncedBabyId = baby.id;
			name = baby.name;
			dateOfBirth = baby.date_of_birth;
			sex = baby.sex;
			diagnosisDate = baby.diagnosis_date ?? '';
			kasaiDate = baby.kasai_date ?? '';
			defaultCalPerFeed = String(baby.default_cal_per_feed ?? 67);
			notes = baby.notes ?? '';
			gestationalWeeks = baby.gestational_age_weeks != null ? String(baby.gestational_age_weeks) : '';
			gestationalDays = baby.gestational_age_days != null ? String(baby.gestational_age_days) : '';
			recalculate = false;
			validationError = '';
		}
	});

	let originalCalPerFeed = $derived(String(baby.default_cal_per_feed ?? 67));

	let calChanged = $derived(defaultCalPerFeed !== originalCalPerFeed);

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!name.trim()) {
			validationError = 'Name is required';
			return;
		}

		const weeksTrim = String(gestationalWeeks ?? '').trim();
		const daysTrim = String(gestationalDays ?? '').trim();
		let weeksValue: number | null = null;
		let daysValue: number | null = null;
		if (weeksTrim !== '') {
			const w = Number(weeksTrim);
			if (!Number.isInteger(w) || w < 20 || w > 44) {
				validationError = 'Gestational weeks must be a whole number between 20 and 44';
				return;
			}
			weeksValue = w;
		}
		if (daysTrim !== '') {
			const d = Number(daysTrim);
			if (!Number.isInteger(d) || d < 0 || d > 6) {
				validationError = 'Gestational days must be a whole number between 0 and 6';
				return;
			}
			daysValue = d;
		}
		if (daysValue !== null && weeksValue === null) {
			validationError = 'Set gestational weeks before days';
			return;
		}

		validationError = '';
		const data: UpdateBabyInput = {
			name: name.trim(),
			date_of_birth: dateOfBirth,
			sex,
			diagnosis_date: diagnosisDate || null,
			kasai_date: kasaiDate || null,
			default_cal_per_feed: parseFloat(defaultCalPerFeed),
			notes: notes.trim() || null,
			gestational_age_weeks: weeksValue,
			gestational_age_days: weeksValue === null ? null : daysValue ?? 0
		};

		onsave(data, calChanged && recalculate);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="settings-name">Name</label>
		<input id="settings-name" type="text" bind:value={name} />
	</div>

	<div>
		<label for="settings-dob">Date of birth</label>
		<input id="settings-dob" type="date" bind:value={dateOfBirth} />
	</div>

	<div>
		<label for="settings-sex">Sex</label>
		<select id="settings-sex" bind:value={sex}>
			<option value="male">Male</option>
			<option value="female">Female</option>
		</select>
	</div>

	<div>
		<label for="settings-diagnosis-date">Diagnosis date</label>
		<input id="settings-diagnosis-date" type="date" bind:value={diagnosisDate} />
	</div>

	<div>
		<label for="settings-kasai-date">Kasai date</label>
		<input id="settings-kasai-date" type="date" bind:value={kasaiDate} />
	</div>

	<div>
		<label for="settings-cal">Default cal per feed</label>
		<input id="settings-cal" type="number" step="any" bind:value={defaultCalPerFeed} />
	</div>

	<fieldset>
		<legend>Gestational age at birth (preterm)</legend>
		<p class="hint">Leave blank for full-term. Enables corrected-age display on the dashboard.</p>
		<div class="gest-row">
			<label for="settings-gest-weeks">Weeks</label>
			<input
				id="settings-gest-weeks"
				type="number"
				step="1"
				bind:value={gestationalWeeks}
			/>
			<label for="settings-gest-days">Days</label>
			<input
				id="settings-gest-days"
				type="number"
				step="1"
				bind:value={gestationalDays}
			/>
		</div>
	</fieldset>

	<div>
		<label for="settings-notes">Notes</label>
		<textarea id="settings-notes" bind:value={notes}></textarea>
	</div>

	{#if calChanged}
		<div>
			<label>
				<input type="checkbox" bind:checked={recalculate} />
				Recalculate existing feeding calories
			</label>
		</div>
	{/if}

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Saving...' : 'Save Settings'}
	</button>
</form>

<style>
	fieldset {
		border: 1px solid var(--color-border, #ddd);
		border-radius: var(--radius-md, 8px);
		padding: var(--space-2, 8px) var(--space-3, 12px);
		margin: var(--space-3, 12px) 0;
	}
	legend {
		padding: 0 var(--space-1, 4px);
		font-weight: 600;
	}
	.hint {
		margin: 0 0 var(--space-2, 8px);
		font-size: var(--font-size-xs, 0.8rem);
		color: var(--color-text-muted, #666);
	}
	.gest-row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2, 8px);
		align-items: center;
	}
	.gest-row input[type='number'] {
		width: 5rem;
	}
</style>
