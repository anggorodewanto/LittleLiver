<script lang="ts">
	import ConfirmAction from './ConfirmAction.svelte';

	interface Props {
		ondelete: () => void;
		error?: string;
	}

	let { ondelete, error = '' }: Props = $props();
</script>

<ConfirmAction
	title="Delete Account"
	message="Are you sure you want to delete your account? This action cannot be undone. All your data will be permanently removed."
	triggerLabel="Delete Account"
	confirmLabel="Confirm Delete"
	onconfirm={ondelete}
	{error}
/>
