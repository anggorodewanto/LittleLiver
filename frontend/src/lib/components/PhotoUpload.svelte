<script lang="ts">
	interface Props {
		id?: string;
		onupload: (file: File) => void;
		uploading?: boolean;
		photoKey?: string;
		multiple?: boolean;
		hint?: string;
		disabled?: boolean;
	}

	let { id = `photo-upload-${Math.random().toString(36).slice(2, 9)}`, onupload, uploading = false, photoKey = '', multiple = false, hint = '', disabled = false }: Props = $props();

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

	<label for={id}>Photo</label>
	<input
		id={id}
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
