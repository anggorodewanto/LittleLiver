<script lang="ts">
	import type { ContainerKind, DoseUnit, MedicationContainer } from '$lib/types/medication';

	export interface ContainerPayload {
		kind: ContainerKind;
		unit: DoseUnit;
		quantity_initial: number;
		quantity_remaining?: number;
		opened_at?: string;
		max_days_after_opening?: number;
		expiration_date?: string;
		depleted?: boolean;
		notes?: string;
	}

	interface Props {
		onsubmit: (payload: ContainerPayload) => void;
		initial?: MedicationContainer | null;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, initial = null, submitting = false, error = '' }: Props = $props();

	let kind = $state<ContainerKind>('bottle');
	let unit = $state<DoseUnit>('ml');
	let quantityInitial = $state<number | undefined>(undefined);
	let quantityRemaining = $state<number | undefined>(undefined);
	let openedAt = $state('');
	let maxDays = $state<number | undefined>(undefined);
	let expirationDate = $state('');
	let depleted = $state(false);
	let notes = $state('');
	let validationError = $state('');

	$effect(() => {
		kind = (initial?.kind as ContainerKind) ?? 'bottle';
		unit = (initial?.unit as DoseUnit) ?? 'ml';
		quantityInitial = initial?.quantity_initial;
		quantityRemaining = initial?.quantity_remaining;
		openedAt = initial?.opened_at ? toLocalInputValue(initial.opened_at) : '';
		maxDays = initial?.max_days_after_opening ?? undefined;
		expirationDate = initial?.expiration_date ?? '';
		depleted = initial?.depleted ?? false;
		notes = initial?.notes ?? '';
		validationError = '';
	});

	function toLocalInputValue(iso: string): string {
		// datetime-local expects YYYY-MM-DDTHH:mm; trim seconds and Z.
		return iso.replace(/Z$/, '').slice(0, 16);
	}

	function fromLocalInputValue(local: string): string {
		// User picked a local time; serialize as ISO UTC.
		const dt = new Date(local);
		return dt.toISOString().replace(/\.\d{3}Z$/, 'Z');
	}

	function handleSubmit(event: SubmitEvent): void {
		event.preventDefault();
		if (quantityInitial === undefined || Number.isNaN(quantityInitial) || quantityInitial < 0) {
			validationError = 'Quantity must be a non-negative number';
			return;
		}
		validationError = '';
		const payload: ContainerPayload = {
			kind,
			unit,
			quantity_initial: quantityInitial
		};
		if (quantityRemaining !== undefined && !Number.isNaN(quantityRemaining)) {
			payload.quantity_remaining = quantityRemaining;
		}
		if (openedAt) {
			payload.opened_at = fromLocalInputValue(openedAt);
		}
		if (maxDays !== undefined && !Number.isNaN(maxDays)) {
			payload.max_days_after_opening = maxDays;
		}
		if (expirationDate) {
			payload.expiration_date = expirationDate;
		}
		if (initial) {
			payload.depleted = depleted;
		}
		if (notes.trim()) {
			payload.notes = notes.trim();
		}
		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="container-kind">Container kind</label>
		<select id="container-kind" bind:value={kind}>
			<option value="bottle">Bottle</option>
			<option value="pill_pack">Pill pack</option>
			<option value="packet">Packet</option>
			<option value="vial">Vial</option>
			<option value="other">Other</option>
		</select>
	</div>

	<div>
		<label for="container-unit">Unit</label>
		<select id="container-unit" bind:value={unit}>
			<option value="ml">mL</option>
			<option value="mg">mg</option>
			<option value="tablet">tablet</option>
			<option value="packet">packet</option>
			<option value="dose">dose</option>
		</select>
	</div>

	<div>
		<label for="container-qty-initial">Quantity (initial)</label>
		<input
			id="container-qty-initial"
			type="number"
			min="0"
			step="any"
			bind:value={quantityInitial}
		/>
	</div>

	{#if initial}
		<div>
			<label for="container-qty-remaining">Quantity remaining</label>
			<input
				id="container-qty-remaining"
				type="number"
				min="0"
				step="any"
				bind:value={quantityRemaining}
			/>
		</div>

		<div>
			<label>
				<input type="checkbox" bind:checked={depleted} />
				Mark as depleted
			</label>
		</div>
	{/if}

	<div>
		<label for="container-opened-at">Opened at (optional — leave blank if sealed)</label>
		<input id="container-opened-at" type="datetime-local" bind:value={openedAt} />
	</div>

	<div>
		<label for="container-max-days">Max days after opening</label>
		<input
			id="container-max-days"
			type="number"
			min="0"
			bind:value={maxDays}
		/>
	</div>

	<div>
		<label for="container-expiration">Manufacturer expiration date</label>
		<input id="container-expiration" type="date" bind:value={expirationDate} />
	</div>

	<div>
		<label for="container-notes">Notes</label>
		<textarea id="container-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}
	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Saving…' : initial ? 'Save changes' : 'Add container'}
	</button>
</form>

<style>
	form > div {
		margin-bottom: 0.5rem;
	}
</style>
