<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';
	import PhotoUpload from './PhotoUpload.svelte';
	import PhotoThumbnails from './PhotoThumbnails.svelte';
	import PhotoLightbox from './PhotoLightbox.svelte';

	export interface NotesPayload {
		timestamp: string;
		content: string;
		category?: string;
		photo_keys?: string[];
	}

	export interface NotesInitialData {
		timestamp: string;
		content: string;
		category?: string;
	}

	interface PhotoInfo {
		key: string;
		url: string;
		thumbnail_url: string;
	}

	interface Props {
		onsubmit: (data: NotesPayload) => void;
		onphotoupload: (file: File) => void;
		onphotoremove?: (key: string) => void;
		initialData?: NotesInitialData;
		submitting?: boolean;
		error?: string;
		uploading?: boolean;
		photoKeys?: string[];
		existingPhotos?: PhotoInfo[];
	}

	let { onsubmit, onphotoupload, onphotoremove, initialData, submitting = false, error = '', uploading = false, photoKeys = [], existingPhotos = [] }: Props = $props();

	let lightboxUrl = $state('');

	let timestamp = $state(defaultTimestamp());
	let content = $state('');
	let category = $state('');
	let validationError = $state('');

	$effect(() => {
		timestamp = initialData ? fromISO8601(initialData.timestamp) : defaultTimestamp();
		content = initialData?.content ?? '';
		category = initialData?.category ?? '';
		validationError = '';
	});

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!content.trim()) {
			validationError = 'Content is required';
			return;
		}

		validationError = '';
		const payload: NotesPayload = {
			timestamp: toISO8601(timestamp),
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

	{#if existingPhotos.length > 0}
		<PhotoThumbnails
			photos={existingPhotos}
			removable={true}
			onremove={onphotoremove}
			onphotoclick={(url) => { lightboxUrl = url; }}
		/>
	{/if}

	{#if lightboxUrl}
		<PhotoLightbox url={lightboxUrl} onclose={() => { lightboxUrl = ''; }} />
	{/if}

	<PhotoUpload onupload={onphotoupload} {uploading} multiple={true} disabled={photoKeys.length >= 4} />

	<p>{photoKeys.length} / 4 photos</p>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting || uploading}>
		{#if uploading}
			Uploading photo...
		{:else if submitting}
			Logging...
		{:else}
			{initialData ? 'Update Note' : 'Log Note'}
		{/if}
	</button>
</form>
