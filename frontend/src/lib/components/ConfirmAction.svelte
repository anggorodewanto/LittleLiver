<script lang="ts">
	interface Props {
		title: string;
		message: string;
		triggerLabel: string;
		confirmLabel: string;
		onconfirm: () => void;
		error?: string;
	}

	let { title, message, triggerLabel, confirmLabel, onconfirm, error = '' }: Props = $props();

	let showConfirmation = $state(false);

	function handleTriggerClick() {
		showConfirmation = true;
	}

	function handleConfirm() {
		showConfirmation = false;
		onconfirm();
	}

	function handleCancel() {
		showConfirmation = false;
	}
</script>

<section>
	<h3>{title}</h3>

	{#if showConfirmation}
		<div role="dialog">
			<p>{message}</p>
			<button onclick={handleConfirm}>{confirmLabel}</button>
			<button onclick={handleCancel}>Cancel</button>
		</div>
	{:else}
		<button onclick={handleTriggerClick}>{triggerLabel}</button>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}
</section>
