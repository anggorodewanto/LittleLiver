<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';

	export interface UrinePayload {
		timestamp: string;
		color?: string;
		volume_ml?: number;
		notes?: string;
	}

	export interface UrineInitialData {
		timestamp: string;
		color?: string;
		volume_ml?: number;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: UrinePayload) => void;
		initialData?: UrineInitialData;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let color = $state('');
	let volumeMl = $state('');
	let notes = $state('');

	$effect(() => {
		if (initialData) {
			timestamp = fromISO8601(initialData.timestamp);
			color = initialData.color ?? '';
			volumeMl = String(initialData.volume_ml ?? '');
			notes = initialData.notes ?? '';
		}
	});

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		const payload: UrinePayload = { timestamp: toISO8601(timestamp) };

		if (color) {
			payload.color = color;
		}
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
		<label for="urine-timestamp">Timestamp</label>
		<input id="urine-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="urine-color">Color</label>
		<select id="urine-color" bind:value={color}>
			<option value="">Select...</option>
			<option value="clear">Clear</option>
			<option value="pale_yellow">Pale Yellow</option>
			<option value="dark_yellow">Dark Yellow</option>
			<option value="amber">Amber</option>
			<option value="brown">Brown</option>
		</select>
	</div>

	<div>
		<label for="urine-volume">Volume (mL)</label>
		<input id="urine-volume" type="number" step="0.1" min="0" bind:value={volumeMl} placeholder="Optional" />
	</div>

	<div>
		<label for="urine-notes">Notes</label>
		<textarea id="urine-notes" bind:value={notes}></textarea>
	</div>

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : initialData ? 'Update Urine' : 'Log Urine'}
	</button>
</form>
