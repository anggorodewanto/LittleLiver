<script lang="ts">
	import { apiClient } from '$lib/api';
	import LabExtractionReview from './LabExtractionReview.svelte';
	import type { ExtractedItem, ReviewedLabPayload } from './LabExtractionReview.svelte';
	import { toISO8601, defaultTimestamp } from '$lib/datetime';

	interface ExtractionResponse {
		extracted: ExtractedItem[];
		notes: string;
		report_date?: string;
	}

	interface Props {
		babyId: string;
		oncancel: () => void;
		onsaved: () => void;
	}

	let { babyId, oncancel, onsaved }: Props = $props();

	type FlowState = 'select' | 'uploading' | 'extracting' | 'review' | 'saving' | 'error' | 'save-error';

	let state = $state<FlowState>('select');
	let error = $state('');
	let extractedItems = $state<ExtractedItem[]>([]);
	let extractionNotes = $state('');
	let reportDate = $state('');

	async function uploadFile(file: File): Promise<string> {
		const formData = new FormData();
		formData.append('file', file);
		const data = await apiClient.postForm<{ r2_key: string }>(`/babies/${babyId}/upload`, formData);
		return data.r2_key;
	}

	async function handleFileSelect(event: Event) {
		const input = event.target as HTMLInputElement;
		if (!input.files || input.files.length === 0) return;

		error = '';
		state = 'uploading';

		const files = Array.from(input.files);

		let keys: string[];
		try {
			keys = await Promise.all(files.map((file) => uploadFile(file)));
		} catch {
			state = 'error';
			error = 'Photo upload failed';
			return;
		}

		state = 'extracting';

		try {
			const response = await apiClient.post<ExtractionResponse>(
				`/babies/${babyId}/labs/extract`,
				{ photo_keys: keys }
			);
			extractedItems = response.extracted;
			extractionNotes = response.notes;
			reportDate = response.report_date ?? '';
			state = 'review';
		} catch {
			state = 'error';
			error = 'Extraction failed. Please try again.';
		}
	}

	function buildTimestamp(): string {
		if (reportDate) {
			// Use report_date at noon local time as the timestamp
			return toISO8601(`${reportDate}T12:00`);
		}
		return toISO8601(defaultTimestamp());
	}

	async function handleConfirm(items: ReviewedLabPayload[]) {
		// Filter out items with empty test_name (validation)
		const validItems = items.filter((item) => item.test_name.trim() !== '');
		if (validItems.length === 0) {
			oncancel();
			return;
		}

		state = 'saving';
		error = '';

		const timestamp = buildTimestamp();

		try {
			await apiClient.post(`/babies/${babyId}/labs/batch`, {
				items: validItems.map((item) => ({
					timestamp,
					...item
				}))
			});
			onsaved();
		} catch {
			state = 'save-error';
			error = 'Failed to save lab results';
		}
	}

</script>

{#if state === 'select' || state === 'error'}
	<div class="import-section">
		<h2>Import from photo</h2>
		<p>Select photos of your lab report to extract results automatically.</p>

		<label for="lab-photo-input">Photo</label>
		<input
			id="lab-photo-input"
			type="file"
			accept="image/*,.pdf,application/pdf"
			multiple
			onchange={handleFileSelect}
		/>

		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}

		<button type="button" onclick={oncancel}>Cancel</button>
	</div>
{:else if state === 'uploading'}
	<div class="loading-state">
		<p>Uploading photos...</p>
	</div>
{:else if state === 'extracting'}
	<div class="loading-state">
		<p>Extracting lab results...</p>
	</div>
{:else if state === 'review' || state === 'save-error'}
	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}
	<LabExtractionReview
		extracted={extractedItems}
		notes={extractionNotes}
		onconfirm={handleConfirm}
		{oncancel}
		{babyId}
	/>
{:else if state === 'saving'}
	<div class="loading-state">
		<p>Saving lab results...</p>
	</div>
{/if}

<style>
	.import-section {
		display: flex;
		flex-direction: column;
		gap: var(--space-2, 0.5rem);
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--space-6, 3rem) var(--space-3, 1rem);
	}

	.error {
		color: var(--color-danger, #dc3545);
	}
</style>
