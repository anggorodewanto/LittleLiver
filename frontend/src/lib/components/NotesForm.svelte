<script lang="ts">
	import { defaultTimestamp } from '$lib/datetime';
	import PhotoUpload from './PhotoUpload.svelte';

	export interface NotesPayload {
		timestamp: string;
		content: string;
		category?: string;
		photo_keys?: string[];
	}

	interface Props {
		onsubmit: (data: NotesPayload) => void;
		onphotoupload: (file: File) => void;
		submitting?: boolean;
		error?: string;
		uploading?: boolean;
		photoKeys?: string[];
	}

	let { onsubmit, onphotoupload, submitting = false, error = '', uploading = false, photoKeys = [] }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let content = $state('');
	let category = $state('');
	let validationError = $state('');

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!content.trim()) {
			validationError = 'Content is required';
			return;
		}

		validationError = '';
		const payload: NotesPayload = {
			timestamp,
			content: content.trim()
		};

		if (category) {
			payload.category = category;
		}
		if (photoKeys.length > 0) {
			payload.photo_keys = photoKeys;
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="notes-timestamp">Timestamp</label>
		<input id="notes-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="notes-content">Content</label>
		<textarea id="notes-content" bind:value={content}></textarea>
	</div>

	<div>
		<label for="notes-category">Category</label>
		<select id="notes-category" bind:value={category}>
			<option value="">Select...</option>
			<option value="behavior">Behavior</option>
			<option value="sleep">Sleep</option>
			<option value="vomiting">Vomiting</option>
			<option value="irritability">Irritability</option>
			<option value="skin">Skin</option>
			<option value="other">Other</option>
		</select>
	</div>

	<PhotoUpload onupload={onphotoupload} {uploading} multiple={true} disabled={photoKeys.length >= 4} />

	<p>{photoKeys.length} / 4 photos</p>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : 'Log Note'}
	</button>
</form>
