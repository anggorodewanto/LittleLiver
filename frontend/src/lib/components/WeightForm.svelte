<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';

	export interface WeightPayload {
		timestamp: string;
		weight_kg: number;
		measurement_source?: string;
		notes?: string;
	}

	export interface WeightInitialData {
		timestamp: string;
		weight_kg: number;
		measurement_source?: string;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: WeightPayload) => void;
		initialData?: WeightInitialData;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let weightKg = $state('');
	let measurementSource = $state('');
	let notes = $state('');
	let validationError = $state('');

	$effect(() => {
		if (initialData) {
			timestamp = fromISO8601(initialData.timestamp);
			weightKg = String(initialData.weight_kg);
			measurementSource = initialData.measurement_source ?? '';
			notes = initialData.notes ?? '';
		}
	});

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!weightKg) {
			validationError = 'Weight is required';
			return;
		}

		validationError = '';
		const payload: WeightPayload = {
			timestamp: toISO8601(timestamp),
			weight_kg: Number(weightKg)
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
		<label for="weight-timestamp">Timestamp</label>
		<input id="weight-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="weight-value">Weight (kg)</label>
		<input id="weight-value" type="number" step="0.01" min="0" bind:value={weightKg} />
	</div>

	<div>
		<label for="weight-source">Measurement source</label>
		<select id="weight-source" bind:value={measurementSource}>
			<option value="">Select...</option>
			<option value="home_scale">Home Scale</option>
			<option value="clinic">Clinic</option>
		</select>
	</div>

	<div>
		<label for="weight-notes">Notes</label>
		<textarea id="weight-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : initialData ? 'Update Weight' : 'Log Weight'}
	</button>
</form>
