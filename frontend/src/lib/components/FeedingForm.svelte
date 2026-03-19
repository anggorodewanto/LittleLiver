<script lang="ts">
	export interface FeedingPayload {
		timestamp: string;
		feed_type: string;
		volume_ml?: number;
		cal_density?: number;
		duration_min?: number;
		notes?: string;
	}

	interface Props {
		onsubmit: (data: FeedingPayload) => void;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, submitting = false, error = '' }: Props = $props();

	function defaultTimestamp(): string {
		const now = new Date();
		const offset = now.getTimezoneOffset();
		const local = new Date(now.getTime() - offset * 60000);
		return local.toISOString().slice(0, 16);
	}

	let timestamp = $state(defaultTimestamp());
	let feedType = $state('');
	let volumeMl = $state('');
	let calDensity = $state('');
	let durationMin = $state('');
	let notes = $state('');
	let validationError = $state('');

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!feedType) {
			validationError = 'Feed type is required';
			return;
		}

		validationError = '';
		const payload: FeedingPayload = {
			timestamp,
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
		{submitting ? 'Logging...' : 'Log Feeding'}
	</button>
</form>
