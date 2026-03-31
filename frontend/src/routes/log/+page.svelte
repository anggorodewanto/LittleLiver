<script lang="ts">
	import { goto } from '$app/navigation';
	import QuickLogButtons from '$lib/components/QuickLogButtons.svelte';
	import type { MetricType } from '$lib/components/QuickLogButtons.svelte';

	function handleSelect(type: MetricType): void {
		if (type === 'med_given') {
			void goto('/log/med');
			return;
		}
		void goto(`/log/${type}`);
	}

	function handleNavigate(path: string): void {
		void goto(path);
	}
</script>

<a href="/" class="back-link">&larr; Back</a>

<h1>What to Log</h1>

<QuickLogButtons onselect={handleSelect} onnavigate={handleNavigate} />

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
