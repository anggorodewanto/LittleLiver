<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';
	import { COLOR_SWATCHES } from '$lib/stool-colors';
	import PhotoUpload from './PhotoUpload.svelte';
	import PhotoThumbnails from './PhotoThumbnails.svelte';
	import PhotoLightbox from './PhotoLightbox.svelte';

	export interface StoolPayload {
		timestamp: string;
		color_rating: number;
		color_label: string;
		consistency?: string;
		volume_estimate?: string;
		volume_ml?: number;
		photo_keys?: string[];
		notes?: string;
	}

	export interface StoolInitialData {
		timestamp: string;
		color_rating: number;
		consistency?: string;
		volume_estimate?: string;
		volume_ml?: number;
		notes?: string;
	}

	interface PhotoInfo {
		key: string;
		url: string;
		thumbnail_url: string;
	}

	interface Props {
		onsubmit: (data: StoolPayload) => void;
		onphotoupload: (file: File) => void;
		onphotoremove?: (key: string) => void;
		initialData?: StoolInitialData;
		submitting?: boolean;
		error?: string;
		uploading?: boolean;
		photoKeys?: string[];
		existingPhotos?: PhotoInfo[];
	}

	let { onsubmit, onphotoupload, onphotoremove, initialData, submitting = false, error = '', uploading = false, photoKeys = [], existingPhotos = [] }: Props = $props();

	let lightboxUrl = $state('');

	let timestamp = $state(defaultTimestamp());
	let colorRating = $state(0);
	let colorLabel = $derived(COLOR_SWATCHES.find(s => s.rating === colorRating)?.ref ?? '');
	let consistency = $state('');
	let volumeEstimate = $state('');
	let volumeMl = $state('');
	let notes = $state('');
	let validationError = $state('');

	$effect(() => {
		if (initialData) {
			timestamp = fromISO8601(initialData.timestamp);
			colorRating = initialData.color_rating;
			consistency = initialData.consistency ?? '';
			volumeEstimate = initialData.volume_estimate ?? '';
			volumeMl = String(initialData.volume_ml ?? '');
			notes = initialData.notes ?? '';
		}
	});

	function selectColor(rating: number) {
		colorRating = rating;
	}

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!colorRating) {
			validationError = 'Stool color is required';
			return;
		}

		validationError = '';
		const payload: StoolPayload = {
			timestamp: toISO8601(timestamp),
			color_rating: colorRating,
			color_label: colorLabel
		};

		if (consistency) {
			payload.consistency = consistency;
		}
		if (volumeEstimate) {
			payload.volume_estimate = volumeEstimate;
		}
		if (volumeMl !== '' && !isNaN(Number(volumeMl))) {
			payload.volume_ml = Number(volumeMl);
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
		<label for="stool-timestamp">Timestamp</label>
		<input id="stool-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<fieldset>
		<legend>Stool Color</legend>
		<div class="color-swatches">
			{#each COLOR_SWATCHES as swatch (swatch.rating)}
				<button
					type="button"
					class="color-swatch"
					aria-pressed={colorRating === swatch.rating ? 'true' : 'false'}
					style="background-color: {swatch.color};{swatch.textColor ? ` color: ${swatch.textColor}` : ''}"
					onclick={() => selectColor(swatch.rating)}
				>
					<span class="swatch-label">{swatch.label}</span>
					<span class="swatch-meaning">{swatch.meaning}</span>
				</button>
			{/each}
		</div>
	</fieldset>

	{#if colorRating >= 1 && colorRating <= 3}
		<p role="alert">
			Warning: Acholic stool detected (color {colorRating}). Contact your hepatology team.
		</p>
	{/if}

	<div>
		<label for="stool-consistency">Consistency</label>
		<select id="stool-consistency" bind:value={consistency}>
			<option value="">Select...</option>
			<option value="watery">Watery</option>
			<option value="loose">Loose</option>
			<option value="soft">Soft</option>
			<option value="formed">Formed</option>
			<option value="hard">Hard</option>
		</select>
	</div>

	<div>
		<label for="stool-volume">Volume estimate</label>
		<select id="stool-volume" bind:value={volumeEstimate}>
			<option value="">Select...</option>
			<option value="small">Small</option>
			<option value="medium">Medium</option>
			<option value="large">Large</option>
		</select>
	</div>

	<div>
		<label for="stool-volume-ml">Volume (mL)</label>
		<input id="stool-volume-ml" type="number" step="0.1" min="0" bind:value={volumeMl} placeholder="Optional" />
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
		<label for="stool-notes">Notes</label>
		<textarea id="stool-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : initialData ? 'Update Stool' : 'Log Stool'}
	</button>
</form>

<style>
	.color-swatches {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
		gap: var(--space-2);
	}

	.color-swatch {
		aspect-ratio: 1;
		border-radius: var(--radius-md);
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--space-1);
		border: 2px solid var(--color-border);
		cursor: pointer;
		min-height: auto;
		transition: border-color 0.15s, box-shadow 0.15s;
	}

	.color-swatch[aria-pressed="true"] {
		border: 3px solid var(--color-text);
		box-shadow: var(--shadow-md);
	}

	.color-swatch:hover {
		background: inherit !important;
		border-color: var(--color-text-muted);
	}

	.swatch-label {
		font-weight: 600;
		font-size: var(--font-size-xs);
	}

	.swatch-meaning {
		font-size: 0.6rem;
		opacity: 0.8;
		text-align: center;
	}
</style>
