<script lang="ts">
	interface Props {
		id?: string;
		onupload: (file: File) => void;
		uploading?: boolean;
		photoKey?: string;
		multiple?: boolean;
		hint?: string;
		disabled?: boolean;
		maxPhotos?: number;
		currentCount?: number;
	}

	const autoId = Math.random().toString(36).slice(2, 9);

	let {
		id = `photo-upload-${autoId}`,
		onupload,
		uploading = false,
		photoKey = '',
		multiple = false,
		hint = '',
		disabled = false,
		maxPhotos = 4,
		currentCount = 0
	}: Props = $props();

	let cameraId = $derived(`${id}-camera`);

	let warning = $state('');

	let isDisabled = $derived(disabled || currentCount >= maxPhotos);

	function handleChange(event: Event) {
		warning = '';
		const input = event.target as HTMLInputElement;
		if (!input.files || input.files.length === 0) {
			return;
		}

		const remaining = maxPhotos - currentCount;
		const filesToUpload = Math.min(input.files.length, remaining);

		if (input.files.length > remaining) {
			warning = `Only ${remaining} of ${input.files.length} photos uploaded (limit: ${maxPhotos})`;
		}

		for (let i = 0; i < filesToUpload; i++) {
			onupload(input.files[i]);
		}

		input.value = '';
	}
</script>

<div>
	{#if hint}
		<p>{hint}</p>
	{/if}

	{#if currentCount > 0}
		<p>{currentCount}/{maxPhotos} photos</p>
	{/if}

	<div class="upload-actions">
		<label for={cameraId} class="upload-button" class:disabled={isDisabled}>Take Photo</label>
		<input
			id={cameraId}
			type="file"
			accept="image/*"
			capture="environment"
			disabled={isDisabled}
			onchange={handleChange}
		/>

		<label for={id} class="upload-button" class:disabled={isDisabled}>Choose Photo</label>
		<input
			id={id}
			type="file"
			accept="image/*"
			{multiple}
			disabled={isDisabled}
			onchange={handleChange}
		/>
	</div>

	{#if uploading}
		<p>Uploading...</p>
	{/if}

	{#if warning}
		<p>{warning}</p>
	{/if}

	{#if photoKey}
		<p>Photo attached</p>
	{:else if currentCount > 0}
		<p>{currentCount} photos attached</p>
	{/if}
</div>

<style>
	.upload-actions {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2, 0.5rem);
	}

	.upload-actions input[type='file'] {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}

	.upload-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-height: 44px;
		padding: var(--space-2, 0.5rem) var(--space-3, 1rem);
		font-size: var(--font-size-sm, 0.875rem);
		font-weight: 600;
		background: var(--color-primary, #0d6efd);
		color: var(--color-text-inverse, #fff);
		border: 1px solid var(--color-primary, #0d6efd);
		border-radius: var(--radius-md, 0.375rem);
		cursor: pointer;
		user-select: none;
	}

	.upload-button:hover {
		background: var(--color-primary-dark, #0a58ca);
		border-color: var(--color-primary-dark, #0a58ca);
	}

	.upload-button.disabled {
		opacity: 0.5;
		cursor: not-allowed;
		pointer-events: none;
	}
</style>
