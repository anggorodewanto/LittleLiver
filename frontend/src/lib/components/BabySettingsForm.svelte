<script lang="ts">
	import type { Baby, UpdateBabyInput } from '$lib/stores/baby';

	interface Props {
		baby: Baby;
		onsave: (data: UpdateBabyInput, recalculate: boolean) => void;
		submitting?: boolean;
		error?: string;
	}

	let { baby, onsave, submitting = false, error = '' }: Props = $props();

	let name = $state(baby.name);
	let dateOfBirth = $state(baby.date_of_birth);
	let sex: 'male' | 'female' = $state(baby.sex);
	let diagnosisDate = $state(baby.diagnosis_date ?? '');
	let kasaiDate = $state(baby.kasai_date ?? '');
	let defaultCalPerFeed = $state(String(baby.default_cal_per_feed ?? 67));
	let recalculate = $state(false);
	let validationError = $state('');

	const originalCalPerFeed = String(baby.default_cal_per_feed ?? 67);

	let calChanged = $derived(defaultCalPerFeed !== originalCalPerFeed);

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!name.trim()) {
			validationError = 'Name is required';
			return;
		}

		validationError = '';
		const data: UpdateBabyInput = {
			name: name.trim(),
			date_of_birth: dateOfBirth,
			sex,
			diagnosis_date: diagnosisDate || null,
			kasai_date: kasaiDate || null,
			default_cal_per_feed: parseFloat(defaultCalPerFeed)
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
