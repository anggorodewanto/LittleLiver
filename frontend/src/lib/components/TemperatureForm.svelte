<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';

	export interface TemperaturePayload {
		timestamp: string;
		value: number;
		method: string;
		notes?: string;
	}

	export interface TemperatureInitialData {
		timestamp: string;
		value: number;
		method: string;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: TemperaturePayload) => void;
		initialData?: TemperatureInitialData;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let value = $state('');
	let method = $state('');
	let notes = $state('');
	let validationError = $state('');

	$effect(() => {
		if (initialData) {
			timestamp = fromISO8601(initialData.timestamp);
			value = String(initialData.value);
			method = initialData.method;
			notes = initialData.notes ?? '';
		}
	});

	const FEVER_THRESHOLDS: Record<string, number> = {
		rectal: 38.0,
		axillary: 37.5,
		ear: 38.0,
		forehead: 37.5
	};

	let feverWarning = $derived.by(() => {
		if (!value || !method) return '';
		const threshold = FEVER_THRESHOLDS[method];
		if (threshold && Number(value) >= threshold) {
			return 'Fever detected. Contact your hepatology team immediately. Fever after Kasai can indicate cholangitis.';
		}
		return '';
	});

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!value) {
			validationError = 'Temperature is required';
			return;
		}

		if (!method) {
			validationError = 'Method is required';
			return;
		}

		validationError = '';
		const payload: TemperaturePayload = {
			timestamp: toISO8601(timestamp),
			value: Number(value),
			method
		};

		if (notes.trim()) {
			payload.notes = notes.trim();
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="temp-timestamp">Timestamp</label>
		<input id="temp-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="temp-value">Temperature (&deg;C)</label>
		<input id="temp-value" type="number" step="0.1" min="30" max="45" bind:value={value} />
	</div>

	<div>
		<label for="temp-method">Method</label>
		<select id="temp-method" bind:value={method}>
			<option value="">Select...</option>
			<option value="rectal">Rectal</option>
			<option value="axillary">Axillary</option>
			<option value="ear">Ear</option>
			<option value="forehead">Forehead</option>
		</select>
	</div>

	<div>
		<label for="temp-notes">Notes</label>
		<textarea id="temp-notes" bind:value={notes}></textarea>
	</div>

	{#if feverWarning}
		<p role="alert" class="fever-warning">{feverWarning}</p>
	{/if}

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : initialData ? 'Update Temperature' : 'Log Temperature'}
	</button>
</form>
