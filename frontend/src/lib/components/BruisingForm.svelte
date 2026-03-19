<script lang="ts">
	import { defaultTimestamp } from '$lib/datetime';
	import PhotoUpload from './PhotoUpload.svelte';

	export interface BruisingPayload {
		timestamp: string;
		location: string;
		size_estimate: string;
		size_cm?: number;
		color?: string;
		photo_keys?: string[];
		notes?: string;
	}

	interface Props {
		onsubmit: (data: BruisingPayload) => void;
		onphotoupload: (file: File) => void;
		submitting?: boolean;
		error?: string;
		uploading?: boolean;
		photoKey?: string;
	}

	let { onsubmit, onphotoupload, submitting = false, error = '', uploading = false, photoKey = '' }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let location = $state('');
	let sizeEstimate = $state('');
	let sizeCm = $state('');
	let color = $state('');
	let notes = $state('');
	let validationError = $state('');

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!location.trim()) {
			validationError = 'Location is required';
			return;
		}

		if (!sizeEstimate) {
			validationError = 'Size estimate is required';
			return;
		}

		validationError = '';
		const payload: BruisingPayload = {
			timestamp,
			location: location.trim(),
			size_estimate: sizeEstimate
		};

		if (sizeCm) {
			payload.size_cm = Number(sizeCm);
		}
		if (color.trim()) {
			payload.color = color.trim();
		}
		if (photoKey) {
			payload.photo_keys = [photoKey];
		}
		if (notes.trim()) {
			payload.notes = notes.trim();
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="bruising-timestamp">Timestamp</label>
		<input id="bruising-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="bruising-location">Location on body</label>
		<input id="bruising-location" type="text" bind:value={location} placeholder="e.g., left arm, torso" />
	</div>

	<div>
		<label for="bruising-size-estimate">Size estimate</label>
		<select id="bruising-size-estimate" bind:value={sizeEstimate}>
			<option value="">Select...</option>
			<option value="small_<1cm">Small (&lt;1cm)</option>
			<option value="medium_1-3cm">Medium (1-3cm)</option>
			<option value="large_>3cm">Large (&gt;3cm)</option>
		</select>
	</div>

	<div>
		<label for="bruising-size-cm">Size (cm)</label>
		<input id="bruising-size-cm" type="number" step="0.1" min="0" bind:value={sizeCm} />
	</div>

	<div>
		<label for="bruising-color">Color</label>
		<input id="bruising-color" type="text" bind:value={color} placeholder="e.g., red, purple, yellow-green" />
	</div>

	<PhotoUpload onupload={onphotoupload} {uploading} {photoKey} />

	<div>
		<label for="bruising-notes">Notes</label>
		<textarea id="bruising-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : 'Log Bruising'}
	</button>
</form>
