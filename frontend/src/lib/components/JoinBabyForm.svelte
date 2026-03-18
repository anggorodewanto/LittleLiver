<script lang="ts">
	interface Props {
		onjoin: (code: string) => void;
		submitting?: boolean;
		error?: string;
	}

	let { onjoin, submitting = false, error = '' }: Props = $props();

	let code = $state('');
	let validationError = $state('');

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		const trimmed = code.trim();
		if (!trimmed) {
			validationError = 'Invite code is required';
			return;
		}

		validationError = '';
		onjoin(trimmed);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="invite-code">Invite code</label>
		<input id="invite-code" type="text" bind:value={code} />
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Joining...' : 'Join'}
	</button>
</form>
