<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';
	import PhotoUpload from './PhotoUpload.svelte';
	import PhotoThumbnails from './PhotoThumbnails.svelte';
	import PhotoLightbox from './PhotoLightbox.svelte';

	export interface BruisingPayload {
		timestamp: string;
		location: string;
		size_estimate: string;
		size_cm?: number;
		color?: string;
		photo_keys?: string[];
		notes?: string;
	}

	export interface BruisingInitialData {
		timestamp: string;
		location: string;
		size_estimate: string;
		size_cm?: number;
		color?: string;
		notes?: string;
	}

	interface PhotoInfo {
		key: string;
		url: string;
		thumbnail_url: string;
	}

	interface Props {
		onsubmit: (data: BruisingPayload) => void;
		onphotoupload: (file: File) => void;
		onphotoremove?: (key: string) => void;
		initialData?: BruisingInitialData;
		submitting?: boolean;
		error?: string;
		uploading?: boolean;
		photoKeys?: string[];
		existingPhotos?: PhotoInfo[];
	}

	let { onsubmit, onphotoupload, onphotoremove, initialData, submitting = false, error = '', uploading = false, photoKeys = [], existingPhotos = [] }: Props = $props();

	let lightboxUrl = $state('');

	let timestamp = $state(defaultTimestamp());
	let location = $state('');
	let sizeEstimate = $state('');
	let sizeCm = $state('');
	let color = $state('');
	let notes = $state('');
	let validationError = $state('');

	$effect(() => {
		if (initialData) {
			timestamp = fromISO8601(initialData.timestamp);
			location = initialData.location;
			sizeEstimate = initialData.size_estimate;
			sizeCm = String(initialData.size_cm ?? '');
			color = initialData.color ?? '';
			notes = initialData.notes ?? '';
		}
	});

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
			timestamp: toISO8601(timestamp),
			location: location.trim(),
			size_estimate: sizeEstimate
		};

		if (sizeCm) {
			payload.size_cm = Number(sizeCm);
		}
		if (color.trim()) {
			payload.color = color.trim();
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
		{submitting ? 'Logging...' : initialData ? 'Update Bruising' : 'Log Bruising'}
	</button>
</form>
