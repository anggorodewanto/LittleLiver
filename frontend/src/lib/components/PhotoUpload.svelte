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

	let { id = `photo-upload-${Math.random().toString(36).slice(2, 9)}`, onupload, uploading = false, photoKey = '', multiple = false, hint = '', disabled = false, maxPhotos = 4, currentCount = 0 }: Props = $props();

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
	}
</script>

<div>
	{#if hint}
		<p>{hint}</p>
	{/if}

	{#if currentCount > 0}
		<p>{currentCount}/{maxPhotos} photos</p>
	{/if}

	<label for={id}>Photo</label>
	<input
		id={id}
		type="file"
		accept="image/jpeg,image/png,image/heic"
		{multiple}
		disabled={isDisabled}
		onchange={handleChange}
	/>

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
