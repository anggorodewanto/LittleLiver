<script lang="ts">
	interface Props {
		oncreate: (data: {
			name: string;
			date_of_birth: string;
			sex: 'male' | 'female';
			diagnosis_date?: string;
			kasai_date?: string;
			gestational_age_weeks?: number | null;
			gestational_age_days?: number | null;
		}) => void;
		submitting?: boolean;
		error?: string;
	}

	let { oncreate, submitting = false, error = '' }: Props = $props();

	let name = $state('');
	let dateOfBirth = $state('');
	let sex: '' | 'male' | 'female' = $state('');
	let diagnosisDate = $state('');
	let kasaiDate = $state('');
	let gestationalWeeks = $state('');
	let gestationalDays = $state('');
	let validationError = $state('');

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		if (!name.trim()) {
			validationError = 'Name is required';
			return;
		}
		if (!dateOfBirth) {
			validationError = 'Date of birth is required';
			return;
		}
		if (!sex) {
			validationError = 'Sex is required';
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
		oncreate({
			name: name.trim(),
			date_of_birth: dateOfBirth,
			sex: sex as 'male' | 'female',
			diagnosis_date: diagnosisDate || undefined,
			kasai_date: kasaiDate || undefined,
			gestational_age_weeks: weeksValue,
			gestational_age_days: weeksValue === null ? null : daysValue ?? 0
		});
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="baby-name">Name</label>
		<input id="baby-name" type="text" bind:value={name} />
	</div>

	<div>
		<label for="baby-dob">Date of birth</label>
		<input id="baby-dob" type="date" bind:value={dateOfBirth} />
	</div>

	<div>
		<label for="baby-sex">Sex</label>
		<select id="baby-sex" bind:value={sex}>
			<option value="">Select...</option>
			<option value="male">Male</option>
			<option value="female">Female</option>
		</select>
	</div>

	<div>
		<label for="baby-diagnosis-date">Diagnosis date</label>
		<input id="baby-diagnosis-date" type="date" bind:value={diagnosisDate} />
	</div>

	<div>
		<label for="baby-kasai-date">Kasai date</label>
		<input id="baby-kasai-date" type="date" bind:value={kasaiDate} />
	</div>

	<fieldset>
		<legend>Gestational age at birth (preterm)</legend>
		<p class="hint">Leave blank for full-term. Enables corrected-age display on the dashboard.</p>
		<div class="gest-row">
			<label for="baby-gest-weeks">Weeks</label>
			<input
				id="baby-gest-weeks"
				type="number"
				step="1"
				bind:value={gestationalWeeks}
			/>
			<label for="baby-gest-days">Days</label>
			<input
				id="baby-gest-days"
				type="number"
				step="1"
				bind:value={gestationalDays}
			/>
		</div>
	</fieldset>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Creating...' : 'Create Baby'}
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
