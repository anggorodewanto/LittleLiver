<script lang="ts">
	import { defaultTimestamp } from '$lib/datetime';
	import PhotoUpload from './PhotoUpload.svelte';

	export interface AbdomenPayload {
		timestamp: string;
		firmness: string;
		tenderness: boolean;
		girth_cm?: number;
		photo_keys?: string[];
		notes?: string;
	}

	interface Props {
		onsubmit: (data: AbdomenPayload) => void;
		onphotoupload: (file: File) => void;
		submitting?: boolean;
		error?: string;
		uploading?: boolean;
		photoKeys?: string[];
	}

	let { onsubmit, onphotoupload, submitting = false, error = '', uploading = false, photoKeys = [] }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let firmness = $state('');
	let tenderness = $state(false);
	let girthCm = $state('');
	let notes = $state('');
	let validationError = $state('');

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!firmness) {
			validationError = 'Firmness is required';
			return;
		}

		validationError = '';
		const payload: AbdomenPayload = {
			timestamp,
			firmness,
			tenderness
		};

		if (girthCm) {
			payload.girth_cm = Number(girthCm);
		}
		if (photoKeys.length > 0) {
			payload.photo_keys = photoKeys;
		}
		if (notes.trim()) {
			payload.notes = notes.trim();
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="abdomen-timestamp">Timestamp</label>
		<input id="abdomen-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="abdomen-firmness">Firmness</label>
		<select id="abdomen-firmness" bind:value={firmness}>
			<option value="">Select...</option>
			<option value="soft">Soft</option>
			<option value="firm">Firm</option>
			<option value="distended">Distended</option>
		</select>
	</div>

	<div>
		<label for="abdomen-tenderness">
			<input id="abdomen-tenderness" type="checkbox" bind:checked={tenderness} />
			Tenderness
		</label>
	</div>

	<div>
		<label for="abdomen-girth">Girth (cm)</label>
		<input id="abdomen-girth" type="number" step="0.1" min="0" bind:value={girthCm} />
	</div>

	<PhotoUpload onupload={onphotoupload} {uploading} multiple={true} currentCount={photoKeys.length} />
	<p>{photoKeys.length} / 4 photos</p>

	<div>
		<label for="abdomen-notes">Notes</label>
		<textarea id="abdomen-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : 'Log Abdomen'}
	</button>
</form>
