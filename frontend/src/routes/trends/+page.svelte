<script lang="ts">
	import { activeBaby } from '$lib/stores/baby';
	import TrendsView from '$lib/components/TrendsView.svelte';

	let baby = $derived($activeBaby);
</script>

{#if !baby}
	<p>No baby selected</p>
{:else}
	<TrendsView babyId={baby.id} sex={baby.sex} dateOfBirth={baby.date_of_birth} />
{/if}
