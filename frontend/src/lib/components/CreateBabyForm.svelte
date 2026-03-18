<script lang="ts">
	interface Props {
		oncreate: (data: {
			name: string;
			date_of_birth: string;
			sex: 'male' | 'female';
			diagnosis_date?: string;
			kasai_date?: string;
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

		validationError = '';
		oncreate({
			name: name.trim(),
			date_of_birth: dateOfBirth,
			sex: sex as 'male' | 'female',
			diagnosis_date: diagnosisDate || undefined,
			kasai_date: kasaiDate || undefined
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
