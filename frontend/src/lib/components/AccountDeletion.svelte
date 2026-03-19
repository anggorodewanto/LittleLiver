<script lang="ts">
	interface Props {
		ondelete: () => void;
		error?: string;
	}

	let { ondelete, error = '' }: Props = $props();

	let showConfirmation = $state(false);

	function handleDeleteClick() {
		showConfirmation = true;
	}

	function handleConfirm() {
		showConfirmation = false;
		ondelete();
	}

	function handleCancel() {
		showConfirmation = false;
	}
</script>

<section>
	<h3>Delete Account</h3>

	{#if showConfirmation}
		<div role="dialog">
			<p>Are you sure you want to delete your account? This action cannot be undone. All your data will be permanently removed.</p>
			<button onclick={handleConfirm}>Confirm Delete</button>
			<button onclick={handleCancel}>Cancel</button>
		</div>
	{:else}
		<button onclick={handleDeleteClick}>Delete Account</button>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}
</section>
