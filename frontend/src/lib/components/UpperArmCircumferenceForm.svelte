<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';

	export interface UpperArmCircumferencePayload {
		timestamp: string;
		circumference_cm: number;
		measurement_source?: string;
		notes?: string;
	}

	export interface UpperArmCircumferenceInitialData {
		timestamp: string;
		circumference_cm: number;
		measurement_source?: string;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: UpperArmCircumferencePayload) => void;
		initialData?: UpperArmCircumferenceInitialData;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let circumferenceCm = $state('');
	let measurementSource = $state('');
	let notes = $state('');
	let validationError = $state('');

	$effect(() => {
		if (initialData) {
			timestamp = fromISO8601(initialData.timestamp);
			circumferenceCm = String(initialData.circumference_cm);
			measurementSource = initialData.measurement_source ?? '';
			notes = initialData.notes ?? '';
		}
	});

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!circumferenceCm) {
			validationError = 'Upper arm circumference is required';
			return;
		}

		validationError = '';
		const payload: UpperArmCircumferencePayload = {
			timestamp: toISO8601(timestamp),
			circumference_cm: Number(circumferenceCm)
		};

		if (measurementSource) {
			payload.measurement_source = measurementSource;
		}
		if (notes.trim()) {
			payload.notes = notes.trim();
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="uac-timestamp">Timestamp</label>
		<input id="uac-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="uac-value">Upper Arm Circumference (cm)</label>
		<input id="uac-value" type="number" step="0.1" min="0" bind:value={circumferenceCm} />
	</div>

	<div>
		<label for="uac-source">Measurement source</label>
		<select id="uac-source" bind:value={measurementSource}>
			<option value="">Select...</option>
			<option value="home_scale">Home</option>
			<option value="clinic">Clinic</option>
		</select>
	</div>

	<div>
		<label for="uac-notes">Notes</label>
		<textarea id="uac-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : initialData ? 'Update Upper Arm Circumference' : 'Log Upper Arm Circumference'}
	</button>
</form>
