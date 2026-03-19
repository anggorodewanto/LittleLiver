<script lang="ts">
	import ConfirmAction from './ConfirmAction.svelte';

	interface Props {
		babyName: string;
		onunlink: () => void;
		error?: string;
	}

	let { babyName, onunlink, error = '' }: Props = $props();
</script>

<ConfirmAction
	title="Unlink from Baby"
	message="Are you sure you want to unlink from {babyName}? If you are the last linked parent, the baby and all associated data will be permanently deleted."
	triggerLabel="Unlink from Baby"
	confirmLabel="Confirm Unlink"
	onconfirm={onunlink}
	{error}
/>
