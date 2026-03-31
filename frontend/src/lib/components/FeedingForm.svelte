<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';

	export interface FeedingPayload {
		timestamp: string;
		feed_type: string;
		volume_ml?: number;
		cal_density?: number;
		duration_min?: number;
		notes?: string;
	}

	export interface FeedingInitialData {
		timestamp: string;
		feed_type: string;
		volume_ml?: number;
		cal_density?: number;
		duration_min?: number;
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
	let validationError = $state('');

	$effect(() => {
		if (initialData) {
			timestamp = fromISO8601(initialData.timestamp);
			feedType = initialData.feed_type;
			volumeMl = String(initialData.volume_ml ?? '');
			calDensity = String(initialData.cal_density ?? '');
			durationMin = String(initialData.duration_min ?? '');
			notes = initialData.notes ?? '';
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

		if (volumeMl) {
			payload.volume_ml = Number(volumeMl);
		}
		if (calDensity) {
			payload.cal_density = Number(calDensity);
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

	<div>
		<label for="feeding-volume">Volume (mL)</label>
		<input id="feeding-volume" type="number" step="any" min="0" bind:value={volumeMl} />
	</div>

	<div>
		<label for="feeding-cal-density">Caloric density (kcal/oz)</label>
		<input id="feeding-cal-density" type="number" step="any" min="0" bind:value={calDensity} />
	</div>

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
