<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';

	export interface HeadCircumferencePayload {
		timestamp: string;
		circumference_cm: number;
		measurement_source?: string;
		notes?: string;
	}

	export interface HeadCircumferenceInitialData {
		timestamp: string;
		circumference_cm: number;
		measurement_source?: string;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: HeadCircumferencePayload) => void;
		initialData?: HeadCircumferenceInitialData;
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
			validationError = 'Head circumference is required';
			return;
		}

		validationError = '';
		const payload: HeadCircumferencePayload = {
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
		<label for="head-circ-timestamp">Timestamp</label>
		<input id="head-circ-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="head-circ-value">Head Circumference (cm)</label>
		<input id="head-circ-value" type="number" step="0.1" min="0" bind:value={circumferenceCm} />
	</div>

	<div>
		<label for="head-circ-source">Measurement source</label>
		<select id="head-circ-source" bind:value={measurementSource}>
			<option value="">Select...</option>
			<option value="home_scale">Home</option>
			<option value="clinic">Clinic</option>
		</select>
	</div>

	<div>
		<label for="head-circ-notes">Notes</label>
		<textarea id="head-circ-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : initialData ? 'Update Head Circumference' : 'Log Head Circumference'}
	</button>
</form>
