<script lang="ts">
	import { getTimeSlotCount } from '$lib/medication-utils';

	export interface MedicationPayload {
		name: string;
		dose: string;
		frequency: string;
		schedule_times: string[];
		interval_days?: number;
		starts_from?: string;
		notes?: string;
		dose_amount?: number;
		dose_unit?: string;
		low_stock_threshold?: number;
		expiry_warning_days?: number;
	}

	export interface MedicationInitialData {
		name: string;
		dose: string;
		frequency: string;
		schedule_times: string[];
		active: boolean;
		interval_days?: number;
		starts_from?: string;
		notes?: string;
		dose_amount?: number | null;
		dose_unit?: string | null;
		low_stock_threshold?: number | null;
		expiry_warning_days?: number | null;
	}

	interface Props {
		onsubmit: (data: MedicationPayload) => void;
		initialData?: MedicationInitialData;
		submitting?: boolean;
		error?: string;
	}

	const MEDICATION_SUGGESTIONS = [
		'UDCA (ursodiol)',
		'Sulfamethoxazole-Trimethoprim (Bactrim)',
		'Vitamin A',
		'Vitamin D',
		'Vitamin E (TPGS)',
		'Vitamin K',
		'Iron',
		'Other'
	];

	let { onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	let name = $state('');
	let dose = $state('');
	let frequency = $state('');
	let scheduleTimes = $state<string[]>([]);
	let intervalDays = $state<number | undefined>(undefined);
	let startsFrom = $state('');
	let notes = $state('');
	let doseAmount = $state<number | undefined>(undefined);
	let doseUnit = $state('');
	let lowStockThreshold = $state<number | undefined>(undefined);
	let expiryWarningDays = $state<number | undefined>(undefined);
	let validationError = $state('');

	$effect(() => {
		name = initialData?.name ?? '';
		dose = initialData?.dose ?? '';
		frequency = initialData?.frequency ?? '';
		scheduleTimes = initialData?.schedule_times ?? [];
		intervalDays = initialData?.interval_days;
		startsFrom = initialData?.starts_from ?? '';
		notes = initialData?.notes ?? '';
		doseAmount = initialData?.dose_amount ?? undefined;
		doseUnit = initialData?.dose_unit ?? '';
		lowStockThreshold = initialData?.low_stock_threshold ?? undefined;
		expiryWarningDays = initialData?.expiry_warning_days ?? undefined;
		validationError = '';
	});

	function handleFrequencyChange(): void {
		if (frequency === 'as_needed') {
			scheduleTimes = [];
			intervalDays = undefined;
			startsFrom = '';
			return;
		}

		if (frequency === 'every_x_days') {
			scheduleTimes = [];
			if (intervalDays === undefined) {
				intervalDays = undefined;
			}
			return;
		}

		intervalDays = undefined;
		startsFrom = '';

		if (frequency === 'custom') {
			if (scheduleTimes.length === 0) {
				scheduleTimes = [''];
			}
			return;
		}

		const count = getTimeSlotCount(frequency);
		if (count > 0) {
			const newTimes: string[] = [];
			for (let i = 0; i < count; i++) {
				newTimes.push(scheduleTimes[i] ?? '');
			}
			scheduleTimes = newTimes;
		}
	}

	function addTime(): void {
		scheduleTimes = [...scheduleTimes, ''];
	}

	function handleSubmit(event: SubmitEvent): void {
		event.preventDefault();

		if (!name.trim()) {
			validationError = 'Medication name is required';
			return;
		}

		if (!dose.trim()) {
			validationError = 'Dose is required';
			return;
		}

		if (!frequency) {
			validationError = 'Frequency is required';
			return;
		}

		if (frequency === 'every_x_days' && (!intervalDays || intervalDays < 1)) {
			validationError = 'Interval days is required and must be at least 1';
			return;
		}

		validationError = '';
		const payload: MedicationPayload = {
			name: name.trim(),
			dose: dose.trim(),
			frequency,
			schedule_times: frequency === 'as_needed' || frequency === 'every_x_days' ? [] : scheduleTimes.filter((t) => t !== '')
		};

		if (frequency === 'every_x_days' && intervalDays) {
			payload.interval_days = intervalDays;
		}

		if (frequency === 'every_x_days' && startsFrom) {
			payload.starts_from = startsFrom;
		}

		if (notes.trim()) {
			payload.notes = notes.trim();
		}

		if (doseAmount !== undefined && !Number.isNaN(doseAmount) && doseAmount > 0) {
			payload.dose_amount = doseAmount;
		}
		if (doseUnit) {
			payload.dose_unit = doseUnit;
		}
		if (lowStockThreshold !== undefined && lowStockThreshold !== null && !Number.isNaN(lowStockThreshold)) {
			payload.low_stock_threshold = lowStockThreshold;
		}
		if (expiryWarningDays !== undefined && expiryWarningDays !== null && !Number.isNaN(expiryWarningDays)) {
			payload.expiry_warning_days = expiryWarningDays;
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="med-name">Medication Name</label>
		<input id="med-name" type="text" list="medication-suggestions" bind:value={name} />
		<datalist id="medication-suggestions">
			{#each MEDICATION_SUGGESTIONS as suggestion (suggestion)}
				<option value={suggestion}></option>
			{/each}
		</datalist>
	</div>

	<div>
		<label for="med-dose">Dose</label>
		<input id="med-dose" type="text" bind:value={dose} />
	</div>

	<div>
		<label for="med-frequency">Frequency</label>
		<select id="med-frequency" bind:value={frequency} onchange={handleFrequencyChange}>
			<option value="">Select...</option>
			<option value="once_daily">Once daily</option>
			<option value="twice_daily">Twice daily</option>
			<option value="three_times_daily">Three times daily</option>
			<option value="every_x_days">Every X days</option>
			<option value="as_needed">As needed</option>
			<option value="custom">Custom</option>
		</select>
	</div>

	{#if frequency === 'every_x_days'}
		<div>
			<label for="med-interval-days">Every how many days</label>
			<input
				id="med-interval-days"
				type="number"
				min="1"
				bind:value={intervalDays}
			/>
		</div>
		<div>
			<label for="med-starts-from">Starts from</label>
			<input
				id="med-starts-from"
				type="date"
				bind:value={startsFrom}
			/>
		</div>
	{/if}

	{#if frequency && frequency !== 'as_needed' && frequency !== 'every_x_days'}
		<!-- eslint-disable-next-line @typescript-eslint/no-unused-vars -->
		{#each scheduleTimes as _time, i (i)}
			<div>
				<label for="med-schedule-time-{i}">Schedule Time {i + 1}</label>
				<input
					id="med-schedule-time-{i}"
					type="time"
					bind:value={scheduleTimes[i]}
				/>
			</div>
		{/each}

		{#if frequency === 'custom'}
			<button type="button" onclick={addTime}>Add Time</button>
		{/if}
	{/if}

	<fieldset>
		<legend>Stock tracking (optional)</legend>
		<p class="hint">Set dose amount + unit to enable auto-decrement when logging doses.</p>
		<div>
			<label for="med-dose-amount">Dose amount per administration</label>
			<input
				id="med-dose-amount"
				type="number"
				min="0"
				step="any"
				bind:value={doseAmount}
			/>
		</div>
		<div>
			<label for="med-dose-unit">Dose unit</label>
			<select id="med-dose-unit" bind:value={doseUnit}>
				<option value="">Select...</option>
				<option value="mg">mg</option>
				<option value="ml">mL</option>
				<option value="tablet">tablet</option>
				<option value="packet">packet</option>
				<option value="dose">dose</option>
			</select>
		</div>
		<div>
			<label for="med-low-stock">Low stock alert (doses left)</label>
			<input
				id="med-low-stock"
				type="number"
				min="0"
				placeholder="default 3"
				bind:value={lowStockThreshold}
			/>
		</div>
		<div>
			<label for="med-expiry-warn">Expiry warning (days before)</label>
			<input
				id="med-expiry-warn"
				type="number"
				min="0"
				placeholder="default 3"
				bind:value={expiryWarningDays}
			/>
		</div>
	</fieldset>

	<div>
		<label for="med-notes">Notes</label>
		<textarea id="med-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Saving...' : 'Save Medication'}
	</button>
</form>
