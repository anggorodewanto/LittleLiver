<script lang="ts">
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { apiClient } from '$lib/api';
	import type { ImmunizationSlot, ImmunizationScheduleResponse } from '$lib/types/immunization';

	interface Props {
		babyId: string;
	}

	let { babyId }: Props = $props();

	let loading = $state(true);
	let error = $state<string | null>(null);
	let slots = $state<ImmunizationSlot[]>([]);

	async function fetchSchedule(): Promise<void> {
		loading = true;
		error = null;
		try {
			const data = await apiClient.get<ImmunizationScheduleResponse>(
				`/babies/${babyId}/immunizations/schedule`
			);
			slots = data.slots ?? [];
		} catch {
			error = 'Failed to load immunizations';
		} finally {
			loading = false;
		}
	}

	function byDueDate(a: ImmunizationSlot, b: ImmunizationSlot): number {
		return (a.due_date ?? '').localeCompare(b.due_date ?? '');
	}

	let onScheduleSlots = $derived(slots.filter((s) => !s.off_schedule));
	let offScheduleSlots = $derived(slots.filter((s) => s.off_schedule));

	let mandatorySlots = $derived(onScheduleSlots.filter((s) => s.mandatory));
	let optionalSlots = $derived(onScheduleSlots.filter((s) => !s.mandatory));

	let mandatoryDoneCount = $derived(mandatorySlots.filter((s) => s.status === 'done').length);

	function pending(group: ImmunizationSlot[]): ImmunizationSlot[] {
		return group.filter((s) => s.status === 'due' || s.status === 'upcoming').sort(byDueDate);
	}

	function completed(group: ImmunizationSlot[]): ImmunizationSlot[] {
		return group.filter((s) => s.status === 'done');
	}

	function handleLog(slot: ImmunizationSlot): void {
		const params = new URLSearchParams({
			code: slot.code,
			name: slot.name,
			dose: String(slot.dose_number)
		});
		goto(`/log/immunization?${params}`);
	}

	function handleEdit(recordId: string): void {
		goto(`/log/immunization?edit=${recordId}`);
	}

	$effect(() => {
		void babyId;
		untrack(() => {
			void fetchSchedule();
		});
	});
</script>

{#if loading}
	<div class="loading">Loading...</div>
{:else if error}
	<div class="error">{error}</div>
{:else if slots.length === 0}
	<div class="empty">No immunizations found.</div>
{:else}
	<p class="summary" data-testid="immunization-summary">
		{mandatoryDoneCount} of {mandatorySlots.length} mandatory done
	</p>

	{#each [{ label: 'Mandatory', group: mandatorySlots }, { label: 'Optional', group: optionalSlots }] as section (section.label)}
		<section class="vax-group">
			<h2>{section.label}</h2>

			{#if pending(section.group).length > 0}
				<h3 class="subsection">Upcoming &amp; due</h3>
				<div class="slot-list">
					{#each pending(section.group) as slot (slot.code + '-' + slot.dose_number)}
						<div class="slot pending" data-testid="immunization-slot">
							<div class="slot-info">
								<span class="slot-name">{slot.name}</span>
								<span class="slot-meta">{slot.dose_label} &middot; {slot.age_label}</span>
								{#if slot.due_date}
									<span class="slot-due">Due {slot.due_date}</span>
								{/if}
							</div>
							<div class="slot-side">
								{#if slot.status === 'due'}
									<span class="badge badge-due">Due</span>
								{:else}
									<span class="badge badge-upcoming">Upcoming</span>
								{/if}
								<button type="button" onclick={() => handleLog(slot)}>Log</button>
							</div>
						</div>
					{/each}
				</div>
			{/if}

			{#if completed(section.group).length > 0}
				<h3 class="subsection">Completed</h3>
				<div class="slot-list">
					{#each completed(section.group) as slot (slot.code + '-' + slot.dose_number)}
						<div class="slot done" data-testid="immunization-slot">
							<div class="slot-info">
								<span class="slot-name">{slot.name}</span>
								<span class="slot-meta">{slot.dose_label} &middot; {slot.age_label}</span>
								{#if slot.administered_date}
									<span class="slot-admin">Given {slot.administered_date}</span>
								{/if}
							</div>
							<div class="slot-side">
								<span class="badge badge-done">Done</span>
								{#if slot.record_id}
									<button type="button" onclick={() => handleEdit(slot.record_id!)}>Edit</button>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</section>
	{/each}

	{#if offScheduleSlots.length > 0}
		<section class="vax-group">
			<h2>Other / off-schedule</h2>
			<div class="slot-list">
				{#each offScheduleSlots as slot (slot.record_id)}
					<div class="slot done" data-testid="immunization-slot">
						<div class="slot-info">
							<span class="slot-name">{slot.name}</span>
							<span class="slot-meta">{slot.dose_label}</span>
							{#if slot.administered_date}
								<span class="slot-admin">Given {slot.administered_date}</span>
							{/if}
						</div>
						<div class="slot-side">
							<span class="badge badge-off">Off-schedule</span>
							{#if slot.record_id}
								<button type="button" onclick={() => handleEdit(slot.record_id!)}>Edit</button>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</section>
	{/if}
{/if}

<style>
	.summary {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		margin: 0 0 var(--space-3);
	}

	.vax-group {
		margin-bottom: var(--space-5);
	}

	.vax-group h2 {
		font-size: var(--font-size-lg);
		margin: 0 0 var(--space-2);
	}

	.subsection {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		margin: var(--space-3) 0 var(--space-2);
	}

	.slot-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.slot {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-3) var(--space-4);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-sm);
	}

	.slot.done {
		opacity: 0.85;
	}

	.slot-info {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		min-width: 0;
	}

	.slot-name {
		font-weight: 600;
	}

	.slot-meta,
	.slot-due,
	.slot-admin {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	.slot-side {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.slot-side button {
		font-size: var(--font-size-xs);
		min-height: 36px;
		padding: var(--space-1) var(--space-3);
	}

	.badge {
		font-size: var(--font-size-xs);
		font-weight: 600;
		padding: 2px var(--space-2);
		border-radius: var(--radius-full);
		white-space: nowrap;
	}

	.badge-due {
		background: var(--color-warning-bg);
		color: var(--color-warning);
	}

	.badge-upcoming {
		background: var(--color-info-bg);
		color: var(--color-info);
	}

	.badge-done {
		background: var(--color-success-bg);
		color: var(--color-success);
	}

	.badge-off {
		background: var(--color-border);
		color: var(--color-text-muted);
	}

	.loading,
	.error,
	.empty {
		text-align: center;
		padding: var(--space-8);
		color: var(--color-text-muted);
	}

	.error {
		color: var(--color-error);
	}
</style>
