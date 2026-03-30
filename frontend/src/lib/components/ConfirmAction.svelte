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

<style>
	div[role="dialog"] {
		background: var(--color-surface);
		border: 1.5px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		margin-top: var(--space-3);
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	div[role="dialog"] button:first-of-type {
		background: var(--color-error);
		color: var(--color-text-inverse);
	}

	div[role="dialog"] button:first-of-type:hover {
		opacity: 0.9;
	}

	div[role="dialog"] button:last-of-type {
		background: var(--color-surface);
		border: 1.5px solid var(--color-border);
		color: var(--color-text);
	}
</style>
