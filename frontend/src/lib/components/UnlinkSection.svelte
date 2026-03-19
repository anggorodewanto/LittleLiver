<script lang="ts">
	interface Props {
		babyName: string;
		onunlink: () => void;
		error?: string;
	}

	let { babyName, onunlink, error = '' }: Props = $props();

	let showConfirmation = $state(false);

	function handleUnlinkClick() {
		showConfirmation = true;
	}

	function handleConfirm() {
		showConfirmation = false;
		onunlink();
	}

	function handleCancel() {
		showConfirmation = false;
	}
</script>

<section>
	<h3>Unlink from Baby</h3>

	{#if showConfirmation}
		<div role="dialog">
			<p>Are you sure you want to unlink from {babyName}? If you are the last linked parent, the baby and all associated data will be permanently deleted.</p>
			<button onclick={handleConfirm}>Confirm Unlink</button>
			<button onclick={handleCancel}>Cancel</button>
		</div>
	{:else}
		<button onclick={handleUnlinkClick}>Unlink from Baby</button>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}
</section>
