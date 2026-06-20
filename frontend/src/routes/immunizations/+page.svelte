<script lang="ts">
	import { activeBaby } from '$lib/stores/baby';
	import ImmunizationView from '$lib/components/ImmunizationView.svelte';

	let baby = $derived($activeBaby);
</script>

<a href="/" class="back-link">&larr; Back</a>

{#if !baby}
	<p>No baby selected</p>
{:else}
	<h1>Immunizations</h1>
	<ImmunizationView babyId={baby.id} />
{/if}

<style>
	.back-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		margin-bottom: var(--space-3);
		min-height: var(--touch-target);
		text-decoration: none;
	}

	.back-link:hover {
		color: var(--color-primary);
	}
</style>
