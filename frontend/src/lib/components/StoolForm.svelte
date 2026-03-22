<script lang="ts">
	import { defaultTimestamp } from '$lib/datetime';
	import { COLOR_SWATCHES } from '$lib/stool-colors';
	import PhotoUpload from './PhotoUpload.svelte';

	export interface StoolPayload {
		timestamp: string;
		color_rating: number;
		color_label: string;
		consistency?: string;
		volume_estimate?: string;
		photo_keys?: string[];
		notes?: string;
	}

	interface Props {
		onsubmit: (data: StoolPayload) => void;
		onphotoupload: (file: File) => void;
		submitting?: boolean;
		error?: string;
		uploading?: boolean;
		photoKeys?: string[];
	}

	let { onsubmit, onphotoupload, submitting = false, error = '', uploading = false, photoKeys = [] }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let colorRating = $state(0);
	let colorLabel = $derived(COLOR_SWATCHES.find(s => s.rating === colorRating)?.ref ?? '');
	let consistency = $state('');
	let volumeEstimate = $state('');
	let notes = $state('');
	let validationError = $state('');

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
			timestamp,
			color_rating: colorRating,
			color_label: colorLabel
		};

		if (consistency) {
			payload.consistency = consistency;
		}
		if (volumeEstimate) {
			payload.volume_estimate = volumeEstimate;
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
		<div style="display: flex; gap: 8px; flex-wrap: wrap;">
			{#each COLOR_SWATCHES as swatch (swatch.rating)}
				<button
					type="button"
					aria-pressed={colorRating === swatch.rating ? 'true' : 'false'}
					style="background-color: {swatch.color}; width: 64px; height: 64px; border: {colorRating === swatch.rating ? '3px solid black' : '1px solid #ccc'}; border-radius: 8px; cursor: pointer; font-size: 0.85rem;"
					onclick={() => selectColor(swatch.rating)}
				>
					{swatch.label}
				</button>
			{/each}
		</div>
	</fieldset>

	{#if colorRating >= 1 && colorRating <= 3}
		<p role="alert" style="color: red; font-weight: bold;">
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
		{submitting ? 'Logging...' : 'Log Stool'}
	</button>
</form>
