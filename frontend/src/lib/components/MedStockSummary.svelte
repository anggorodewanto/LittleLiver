<script lang="ts">
	import { apiClient } from '$lib/api';
	import type { MedicationContainer } from '$lib/types/medication';

	interface Props {
		babyId: string;
		medicationId: string;
		doseAmount?: number | null;
		doseUnit?: string | null;
	}

	let { babyId, medicationId, doseAmount = null, doseUnit = null }: Props = $props();

	let containers = $state<MedicationContainer[]>([]);
	let loaded = $state(false);

	$effect(() => {
		void babyId;
		void medicationId;
		(async () => {
			try {
				containers = await apiClient.get<MedicationContainer[]>(
					`/babies/${babyId}/medications/${medicationId}/containers`
				);
			} catch {
				containers = [];
			} finally {
				loaded = true;
			}
		})();
	});

	let activeContainers = $derived(containers.filter((c) => !c.depleted));
	let totalRemaining = $derived(
		activeContainers.reduce((acc, c) => acc + c.quantity_remaining, 0)
	);
	let totalDoses = $derived(
		doseAmount && doseAmount > 0 ? Math.floor(totalRemaining / doseAmount) : null
	);
</script>

{#if loaded && containers.length > 0}
	<span class="stock-summary" data-testid="stock-summary">
		{activeContainers.length} containers · ~{totalRemaining}{' '}{doseUnit ?? ''}
		{#if totalDoses !== null}
			· ~{totalDoses} doses left
		{/if}
	</span>
{/if}

<style>
	.stock-summary {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}
</style>
