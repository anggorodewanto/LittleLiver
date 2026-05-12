<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';

	export interface HeightPayload {
		timestamp: string;
		height_cm: number;
		measurement_source?: string;
		notes?: string;
	}

	export interface HeightInitialData {
		timestamp: string;
		height_cm: number;
		measurement_source?: string;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: HeightPayload) => void;
		initialData?: HeightInitialData;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let heightCm = $state('');
	let measurementSource = $state('');
	let notes = $state('');
	let validationError = $state('');

	$effect(() => {
		if (initialData) {
			timestamp = fromISO8601(initialData.timestamp);
			heightCm = String(initialData.height_cm);
			measurementSource = initialData.measurement_source ?? '';
			notes = initialData.notes ?? '';
		}
	});

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!heightCm) {
			validationError = 'Height is required';
			return;
		}

		validationError = '';
		const payload: HeightPayload = {
			timestamp: toISO8601(timestamp),
			height_cm: Number(heightCm)
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
		<label for="height-timestamp">Timestamp</label>
		<input id="height-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="height-value">Height (cm)</label>
		<input id="height-value" type="number" step="0.1" min="0" bind:value={heightCm} />
	</div>

	<div>
		<label for="height-source">Measurement source</label>
		<select id="height-source" bind:value={measurementSource}>
			<option value="">Select...</option>
			<option value="home_scale">Home</option>
			<option value="clinic">Clinic</option>
		</select>
	</div>

	<div>
		<label for="height-notes">Notes</label>
		<textarea id="height-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : initialData ? 'Update Height' : 'Log Height'}
	</button>
</form>
