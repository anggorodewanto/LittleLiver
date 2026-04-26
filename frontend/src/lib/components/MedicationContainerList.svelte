<script lang="ts">
	import { apiClient } from '$lib/api';
	import type { ContainerKind, DoseUnit, MedicationContainer } from '$lib/types/medication';
	import MedicationContainerForm, {
		type ContainerPayload
	} from './MedicationContainerForm.svelte';

	interface Props {
		babyId: string;
		medicationId: string;
		doseAmount?: number | null;
		doseUnit?: DoseUnit | null;
	}

	let { babyId, medicationId, doseAmount = null, doseUnit = null }: Props = $props();

	let containers = $state<MedicationContainer[]>([]);
	let loading = $state(true);
	let error = $state('');
	let editingId = $state<string | null>(null);
	let showAdd = $state(false);
	let submitting = $state(false);
	let formError = $state('');
	let adjustingId = $state<string | null>(null);
	let adjustDelta = $state<number | undefined>(undefined);
	let adjustReason = $state('');

	async function load() {
		loading = true;
		try {
			containers = await apiClient.get<MedicationContainer[]>(
				`/babies/${babyId}/medications/${medicationId}/containers`
			);
			error = '';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load containers';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	function totalRemaining(): number {
		return containers
			.filter((c) => !c.depleted)
			.reduce((acc, c) => acc + c.quantity_remaining, 0);
	}

	function totalDoses(): number | null {
		if (!doseAmount || doseAmount <= 0) return null;
		return Math.floor(totalRemaining() / doseAmount);
	}

	async function handleCreate(payload: ContainerPayload) {
		submitting = true;
		formError = '';
		try {
			await apiClient.post(
				`/babies/${babyId}/medications/${medicationId}/containers`,
				payload
			);
			showAdd = false;
			await load();
		} catch (e) {
			formError = e instanceof Error ? e.message : 'Failed to add container';
		} finally {
			submitting = false;
		}
	}

	async function handleUpdate(payload: ContainerPayload, id: string) {
		submitting = true;
		formError = '';
		try {
			await apiClient.put(
				`/babies/${babyId}/medications/${medicationId}/containers/${id}`,
				payload
			);
			editingId = null;
			await load();
		} catch (e) {
			formError = e instanceof Error ? e.message : 'Failed to save container';
		} finally {
			submitting = false;
		}
	}

	async function handleDelete(id: string) {
		if (!confirm('Delete this container? This cannot be undone.')) return;
		try {
			await apiClient.del(
				`/babies/${babyId}/medications/${medicationId}/containers/${id}`
			);
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete container';
		}
	}

	async function handleAdjust(id: string) {
		if (adjustDelta === undefined || Number.isNaN(adjustDelta) || adjustDelta === 0) return;
		try {
			await apiClient.post(
				`/babies/${babyId}/medications/${medicationId}/containers/${id}/adjust`,
				{ delta: adjustDelta, reason: adjustReason || undefined }
			);
			adjustingId = null;
			adjustDelta = undefined;
			adjustReason = '';
			await load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to adjust stock';
		}
	}

	function kindLabel(k: ContainerKind): string {
		const labels: Record<ContainerKind, string> = {
			bottle: 'Bottle',
			pill_pack: 'Pill pack',
			packet: 'Packet',
			vial: 'Vial',
			other: 'Other'
		};
		return labels[k] ?? k;
	}
</script>

<section>
	<h3>Stock containers</h3>

	{#if loading}
		<p>Loading…</p>
	{:else}
		{#if containers.length > 0}
			<p data-testid="stock-summary">
				{containers.length} containers · ~{totalRemaining()}{' '}{doseUnit ?? ''} remaining
				{#if totalDoses() !== null}
					· ~{totalDoses()} doses left
				{/if}
			</p>
		{:else}
			<p>No stock containers yet.</p>
		{/if}

		{#if error}
			<p role="alert">{error}</p>
		{/if}

		<ul>
			{#each containers as c (c.id)}
				<li data-testid="container-row">
					<strong>{kindLabel(c.kind)}</strong> — {c.quantity_remaining}{' '}{c.unit}
					{#if c.depleted}<em>(depleted)</em>{/if}
					{#if c.effective_expiry}
						· effective expiry {c.effective_expiry}
					{/if}

					<button type="button" onclick={() => (editingId = c.id)}>Edit</button>
					<button type="button" onclick={() => (adjustingId = c.id)}>Adjust</button>
					<button type="button" onclick={() => handleDelete(c.id)}>Delete</button>

					{#if editingId === c.id}
						<MedicationContainerForm
							initial={c}
							submitting={submitting}
							error={formError}
							onsubmit={(payload) => handleUpdate(payload, c.id)}
						/>
						<button type="button" onclick={() => (editingId = null)}>Cancel</button>
					{/if}

					{#if adjustingId === c.id}
						<form
							onsubmit={(e) => {
								e.preventDefault();
								handleAdjust(c.id);
							}}
						>
							<label>
								Delta (+ to add, − to remove)
								<input type="number" step="any" bind:value={adjustDelta} />
							</label>
							<label>
								Reason
								<input type="text" bind:value={adjustReason} />
							</label>
							<button type="submit">Apply</button>
							<button type="button" onclick={() => (adjustingId = null)}>Cancel</button>
						</form>
					{/if}
				</li>
			{/each}
		</ul>

		{#if showAdd}
			<MedicationContainerForm
				submitting={submitting}
				error={formError}
				onsubmit={handleCreate}
			/>
			<button type="button" onclick={() => (showAdd = false)}>Cancel</button>
		{:else}
			<button type="button" onclick={() => (showAdd = true)}>+ Add container</button>
		{/if}
	{/if}
</section>
