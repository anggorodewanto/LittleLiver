<script lang="ts">
	import { defaultTimestamp, toISO8601 } from '$lib/datetime';

	export interface UrinePayload {
		timestamp: string;
		color?: string;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: UrinePayload) => void;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, submitting = false, error = '' }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let color = $state('');
	let notes = $state('');

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		const payload: UrinePayload = { timestamp: toISO8601(timestamp) };

		if (color) {
			payload.color = color;
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
		<label for="urine-notes">Notes</label>
		<textarea id="urine-notes" bind:value={notes}></textarea>
	</div>

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : 'Log Urine'}
	</button>
</form>
