<script lang="ts">
	interface Props {
		onupload: (file: File) => void;
		uploading?: boolean;
		photoKey?: string;
		multiple?: boolean;
		hint?: string;
		disabled?: boolean;
	}

	let { onupload, uploading = false, photoKey = '', multiple = false, hint = '', disabled = false }: Props = $props();

	function handleChange(event: Event) {
		const input = event.target as HTMLInputElement;
		if (!input.files || input.files.length === 0) {
			return;
		}
		for (let i = 0; i < input.files.length; i++) {
			onupload(input.files[i]);
		}
	}
</script>

<div>
	{#if hint}
		<p>{hint}</p>
	{/if}

	<label for="photo-upload">Photo</label>
	<input
		id="photo-upload"
		type="file"
		accept="image/jpeg,image/png,image/heic"
		{multiple}
		{disabled}
		onchange={handleChange}
	/>

	{#if uploading}
		<p>Uploading...</p>
	{/if}

	{#if photoKey}
		<p>Photo attached</p>
	{/if}
</div>
