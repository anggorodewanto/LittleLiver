<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';

	export interface FeedingPayload {
		timestamp: string;
		feed_type: string;
		volume_ml?: number;
		cal_density?: number;
		duration_min?: number;
		amount_g?: number;
		ingredients?: string;
		notes?: string;
	}

	export interface FeedingInitialData {
		timestamp: string;
		feed_type: string;
		volume_ml?: number;
		cal_density?: number;
		duration_min?: number;
		amount_g?: number;
		ingredients?: string;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: FeedingPayload) => void;
		initialData?: FeedingInitialData;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	let timestamp = $state(defaultTimestamp());
	let feedType = $state('');
	let volumeMl = $state('');
	let calDensity = $state('');
	let durationMin = $state('');
	let notes = $state('');
	let amount = $state('');
	let amountUnit = $state('g');
	let ingredients = $state('');
	let validationError = $state('');

	let isSolid = $derived(feedType === 'solid');

	$effect(() => {
		if (initialData) {
			timestamp = fromISO8601(initialData.timestamp);
			feedType = initialData.feed_type;
			volumeMl = String(initialData.volume_ml ?? '');
			calDensity = String(initialData.cal_density ?? '');
			durationMin = String(initialData.duration_min ?? '');
			notes = initialData.notes ?? '';
			ingredients = initialData.ingredients ?? '';
			if (initialData.amount_g != null) {
				amount = String(initialData.amount_g);
				amountUnit = 'g';
			} else if (initialData.feed_type === 'solid' && initialData.volume_ml != null) {
				amount = String(initialData.volume_ml);
				amountUnit = 'ml';
			}
		}
	});

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!feedType) {
			validationError = 'Feed type is required';
			return;
		}

		validationError = '';
		const payload: FeedingPayload = {
			timestamp: toISO8601(timestamp),
			feed_type: feedType
		};

		if (isSolid) {
			if (amount) {
				if (amountUnit === 'g') {
					payload.amount_g = Number(amount);
				} else {
					payload.volume_ml = Number(amount);
				}
			}
			if (ingredients.trim()) {
				payload.ingredients = ingredients.trim();
			}
		} else {
			if (volumeMl) {
				payload.volume_ml = Number(volumeMl);
			}
			if (calDensity) {
				payload.cal_density = Number(calDensity);
			}
		}
		if (durationMin) {
			payload.duration_min = Number(durationMin);
		}
		if (notes.trim()) {
			payload.notes = notes.trim();
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="feeding-timestamp">Timestamp</label>
		<input id="feeding-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	<div>
		<label for="feeding-type">Feed type</label>
		<select id="feeding-type" bind:value={feedType}>
			<option value="">Select...</option>
			<option value="breast_milk">Breast Milk</option>
			<option value="formula">Formula</option>
			<option value="fortified_breast_milk">Fortified Breast Milk</option>
			<option value="solid">Solid</option>
			<option value="other">Other</option>
		</select>
	</div>

	{#if isSolid}
		<div>
			<label for="feeding-amount">Amount</label>
			<input id="feeding-amount" type="number" step="any" min="0" bind:value={amount} />
		</div>

		<div>
			<label for="feeding-amount-unit">Unit</label>
			<select id="feeding-amount-unit" bind:value={amountUnit}>
				<option value="g">g</option>
				<option value="ml">mL</option>
			</select>
		</div>

		<div>
			<label for="feeding-ingredients">Ingredients</label>
			<textarea id="feeding-ingredients" bind:value={ingredients} placeholder="rice porridge, carrot, chicken"></textarea>
		</div>
	{:else}
		<div>
			<label for="feeding-volume">Volume (mL)</label>
			<input id="feeding-volume" type="number" step="any" min="0" bind:value={volumeMl} />
		</div>

		<div>
			<label for="feeding-cal-density">Caloric density (kcal/mL)</label>
			<input id="feeding-cal-density" type="number" step="any" min="0" bind:value={calDensity} placeholder="0.676 (default)" />
		</div>
	{/if}

	<div>
		<label for="feeding-duration">Duration (min)</label>
		<input id="feeding-duration" type="number" step="1" min="0" bind:value={durationMin} />
	</div>

	<div>
		<label for="feeding-notes">Notes</label>
		<textarea id="feeding-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Logging...' : initialData ? 'Update Feeding' : 'Log Feeding'}
	</button>
</form>
