<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';

	export interface FluidLogPayload {
		timestamp: string;
		direction: 'intake' | 'output';
		method: string;
		volume_ml?: number;
		notes?: string;
	}

	export interface FluidLogInitialData {
		timestamp: string;
		method: string;
		volume_ml?: number;
		notes?: string;
	}

	interface Props {
		direction: 'intake' | 'output';
		onsubmit: (data: FluidLogPayload) => void;
		initialData?: FluidLogInitialData;
		submitting?: boolean;
		error?: string;
	}

	let { direction, onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let method = $state('');
	let volumeMl = $state('');
	let notes = $state('');
	let validationError = $state('');

	$effect(() => {
		timestamp = initialData ? fromISO8601(initialData.timestamp) : defaultTimestamp();
		method = initialData?.method ?? '';
		volumeMl = String(initialData?.volume_ml ?? '');
		notes = initialData?.notes ?? '';
		validationError = '';
	});

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!method.trim()) {
			validationError = 'Method is required';
			return;
		}

		validationError = '';
		const payload: FluidLogPayload = {
			timestamp: toISO8601(timestamp),
			direction,
			method: method.trim()
		};

		if (volumeMl !== '' && !isNaN(Number(volumeMl))) {
			payload.volume_ml = Number(volumeMl);
		}
		if (notes.trim()) {
			payload.notes = notes.trim();
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="fluid-timestamp">Timestamp</label>
		<input id="fluid-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="fluid-method">Method</label>
		<input id="fluid-method" type="text" bind:value={method} placeholder={direction === 'intake' ? 'e.g., IV, NG tube' : 'e.g., Stoma, Drain'} />
	</div>

	<div>
		<label for="fluid-volume">Volume (mL)</label>
		<input id="fluid-volume" type="number" step="0.1" min="0" bind:value={volumeMl} placeholder="Optional" />
	</div>

	<div>
		<label for="fluid-notes">Notes</label>
		<textarea id="fluid-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : initialData ? `Update ${direction === 'intake' ? 'Intake' : 'Output'}` : `Log ${direction === 'intake' ? 'Intake' : 'Output'}`}
	</button>
</form>
