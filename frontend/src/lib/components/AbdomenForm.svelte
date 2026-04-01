<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';
	import PhotoUpload from './PhotoUpload.svelte';
	import PhotoThumbnails from './PhotoThumbnails.svelte';
	import PhotoLightbox from './PhotoLightbox.svelte';

	export interface AbdomenPayload {
		timestamp: string;
		firmness: string;
		tenderness: boolean;
		girth_cm?: number;
		photo_keys?: string[];
		notes?: string;
	}

	export interface AbdomenInitialData {
		timestamp: string;
		firmness: string;
		tenderness: boolean;
		girth_cm?: number;
		notes?: string;
	}

	interface PhotoInfo {
		key: string;
		url: string;
		thumbnail_url: string;
	}

	interface Props {
		onsubmit: (data: AbdomenPayload) => void;
		onphotoupload: (file: File) => void;
		onphotoremove?: (key: string) => void;
		initialData?: AbdomenInitialData;
		submitting?: boolean;
		error?: string;
		uploading?: boolean;
		photoKeys?: string[];
		existingPhotos?: PhotoInfo[];
	}

	let { onsubmit, onphotoupload, onphotoremove, initialData, submitting = false, error = '', uploading = false, photoKeys = [], existingPhotos = [] }: Props = $props();

	let lightboxUrl = $state('');

	let timestamp = $state(defaultTimestamp());
	let firmness = $state('');
	let tenderness = $state(false);
	let girthCm = $state('');
	let notes = $state('');
	let validationError = $state('');

	$effect(() => {
		if (initialData) {
			timestamp = fromISO8601(initialData.timestamp);
			firmness = initialData.firmness;
			tenderness = initialData.tenderness;
			girthCm = String(initialData.girth_cm ?? '');
			notes = initialData.notes ?? '';
		}
	});

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!firmness) {
			validationError = 'Firmness is required';
			return;
		}

		validationError = '';
		const payload: AbdomenPayload = {
			timestamp: toISO8601(timestamp),
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
		{submitting ? 'Logging...' : initialData ? 'Update Abdomen' : 'Log Abdomen'}
	</button>
</form>
