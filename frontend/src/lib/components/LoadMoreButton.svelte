<script lang="ts">
	interface Props {
		loading?: boolean;
		hasMore: boolean;
		onloadmore: () => void;
	}

	let { loading = false, hasMore, onloadmore }: Props = $props();
</script>

{#if hasMore}
	{#if loading}
		<p>Loading...</p>
	{:else}
		<button onclick={onloadmore}>Load more</button>
	{/if}
{/if}
