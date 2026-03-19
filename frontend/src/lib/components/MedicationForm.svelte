<script lang="ts">
	export interface MedicationPayload {
		name: string;
		dose: string;
		frequency: string;
		schedule_times: string[];
		notes?: string;
	}

	export interface MedicationInitialData {
		name: string;
		dose: string;
		frequency: string;
		schedule_times: string[];
		active: boolean;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: MedicationPayload) => void;
		initialData?: MedicationInitialData;
		submitting?: boolean;
		error?: string;
	}

	const MEDICATION_SUGGESTIONS = [
		'UDCA (ursodiol)',
		'Sulfamethoxazole-Trimethoprim (Bactrim)',
		'Vitamin A',
		'Vitamin D',
		'Vitamin E (TPGS)',
		'Vitamin K',
		'Iron',
		'Other'
	];

	let { onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	let name = $state('');
	let dose = $state('');
	let frequency = $state('');
	let scheduleTimes = $state<string[]>([]);
	let notes = $state('');
	let validationError = $state('');

	$effect(() => {
		name = initialData?.name ?? '';
		dose = initialData?.dose ?? '';
		frequency = initialData?.frequency ?? '';
		scheduleTimes = initialData?.schedule_times ?? [];
		notes = initialData?.notes ?? '';
		validationError = '';
	});

	function getTimeSlotCount(freq: string): number {
		switch (freq) {
			case 'once_daily':
				return 1;
			case 'twice_daily':
				return 2;
			case 'three_times_daily':
				return 3;
			default:
				return 0;
		}
	}

	function handleFrequencyChange(): void {
		if (frequency === 'as_needed') {
			scheduleTimes = [];
			return;
		}

		if (frequency === 'custom') {
			if (scheduleTimes.length === 0) {
				scheduleTimes = [''];
			}
			return;
		}

		const count = getTimeSlotCount(frequency);
		if (count > 0) {
			const newTimes: string[] = [];
			for (let i = 0; i < count; i++) {
				newTimes.push(scheduleTimes[i] ?? '');
			}
			scheduleTimes = newTimes;
		}
	}

	function addTime(): void {
		scheduleTimes = [...scheduleTimes, ''];
	}

	function handleSubmit(event: SubmitEvent): void {
		event.preventDefault();

		if (!name.trim()) {
			validationError = 'Medication name is required';
			return;
		}

		if (!dose.trim()) {
			validationError = 'Dose is required';
			return;
		}

		if (!frequency) {
			validationError = 'Frequency is required';
			return;
		}

		validationError = '';
		const payload: MedicationPayload = {
			name: name.trim(),
			dose: dose.trim(),
			frequency,
			schedule_times: frequency === 'as_needed' ? [] : scheduleTimes.filter((t) => t !== '')
		};

		if (notes.trim()) {
			payload.notes = notes.trim();
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="med-name">Medication Name</label>
		<input id="med-name" type="text" list="medication-suggestions" bind:value={name} />
		<datalist id="medication-suggestions">
			{#each MEDICATION_SUGGESTIONS as suggestion (suggestion)}
				<option value={suggestion}></option>
			{/each}
		</datalist>
	</div>

	<div>
		<label for="med-dose">Dose</label>
		<input id="med-dose" type="text" bind:value={dose} />
	</div>

	<div>
		<label for="med-frequency">Frequency</label>
		<select id="med-frequency" bind:value={frequency} onchange={handleFrequencyChange}>
			<option value="">Select...</option>
			<option value="once_daily">Once daily</option>
			<option value="twice_daily">Twice daily</option>
			<option value="three_times_daily">Three times daily</option>
			<option value="as_needed">As needed</option>
			<option value="custom">Custom</option>
		</select>
	</div>

	{#if frequency && frequency !== 'as_needed'}
		<!-- eslint-disable-next-line @typescript-eslint/no-unused-vars -->
		{#each scheduleTimes as _time, i (i)}
			<div>
				<label for="med-schedule-time-{i}">Schedule Time {i + 1}</label>
				<input
					id="med-schedule-time-{i}"
					type="time"
					bind:value={scheduleTimes[i]}
				/>
			</div>
		{/each}

		{#if frequency === 'custom'}
			<button type="button" onclick={addTime}>Add Time</button>
		{/if}
	{/if}

	<div>
		<label for="med-notes">Notes</label>
		<textarea id="med-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Saving...' : 'Save Medication'}
	</button>
</form>
