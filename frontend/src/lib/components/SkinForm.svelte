<script lang="ts">
	import { defaultTimestamp } from '$lib/datetime';
	import PhotoUpload from './PhotoUpload.svelte';

	export interface SkinPayload {
		timestamp: string;
		jaundice_level?: string;
		scleral_icterus: boolean;
		rashes?: string;
		bruising?: string;
		photo_keys?: string[];
		notes?: string;
	}

	interface Props {
		onsubmit: (data: SkinPayload) => void;
		onphotoupload: (file: File) => void;
		submitting?: boolean;
		error?: string;
		uploading?: boolean;
		photoKey?: string;
	}

	let { onsubmit, onphotoupload, submitting = false, error = '', uploading = false, photoKey = '' }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let jaundiceLevel = $state('');
	let scleralIcterus = $state(false);
	let rashes = $state('');
	let bruising = $state('');
	let notes = $state('');

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		const payload: SkinPayload = {
			timestamp,
			scleral_icterus: scleralIcterus
		};

		if (jaundiceLevel) {
			payload.jaundice_level = jaundiceLevel;
		}
		if (rashes.trim()) {
			payload.rashes = rashes.trim();
		}
		if (bruising.trim()) {
			payload.bruising = bruising.trim();
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
		<label for="skin-timestamp">Timestamp</label>
		<input id="skin-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="skin-jaundice">Jaundice level</label>
		<select id="skin-jaundice" bind:value={jaundiceLevel}>
			<option value="">Select...</option>
			<option value="none">None</option>
			<option value="mild_face">Mild (Face)</option>
			<option value="moderate_trunk">Moderate (Trunk)</option>
			<option value="severe_limbs_and_trunk">Severe (Limbs & Trunk)</option>
		</select>
	</div>

	<div>
		<label for="skin-scleral">
			<input id="skin-scleral" type="checkbox" bind:checked={scleralIcterus} />
			Scleral icterus
		</label>
	</div>

	<div>
		<label for="skin-rashes">Rashes</label>
		<input id="skin-rashes" type="text" bind:value={rashes} />
	</div>

	<div>
		<label for="skin-bruising">Bruising</label>
		<input id="skin-bruising" type="text" bind:value={bruising} />
	</div>

	<PhotoUpload onupload={onphotoupload} {uploading} {photoKey} hint="Consistent lighting recommended" />

	<div>
		<label for="skin-notes">Notes</label>
		<textarea id="skin-notes" bind:value={notes}></textarea>
	</div>

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : 'Log Skin'}
	</button>
</form>
