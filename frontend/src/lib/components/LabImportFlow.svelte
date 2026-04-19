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

	const MAX_FILES = 10;

	let state = $state<FlowState>('select');
	let error = $state('');
	let extractedItems = $state<ExtractedItem[]>([]);
	let extractionNotes = $state('');
	let reportDate = $state('');
	let selectedFiles = $state<File[]>([]);

	async function uploadFile(file: File): Promise<string> {
		const formData = new FormData();
		formData.append('file', file);
		const data = await apiClient.postForm<{ r2_key: string }>(`/babies/${babyId}/upload`, formData);
		return data.r2_key;
	}

	function handleFileAdd(event: Event) {
		const input = event.target as HTMLInputElement;
		if (!input.files || input.files.length === 0) return;

		const incoming = Array.from(input.files);
		const remaining = MAX_FILES - selectedFiles.length;
		selectedFiles = [...selectedFiles, ...incoming.slice(0, remaining)];

		input.value = '';
	}

	function removeFile(index: number) {
		selectedFiles = selectedFiles.filter((_, i) => i !== index);
	}

	async function handleSubmit() {
		if (selectedFiles.length === 0) return;

		error = '';
		state = 'uploading';

		let keys: string[];
		try {
			keys = await Promise.all(selectedFiles.map((file) => uploadFile(file)));
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
			return toISO8601(`${reportDate}T12:00`);
		}
		return toISO8601(defaultTimestamp());
	}

	async function handleConfirm(items: ReviewedLabPayload[]) {
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
			onchange={handleFileAdd}
		/>

		{#if selectedFiles.length > 0}
			<ul class="queued-files">
				{#each selectedFiles as file, i (i + '-' + file.name)}
					<li>
						<span>{file.name}</span>
						<button type="button" aria-label="Remove {file.name}" onclick={() => removeFile(i)}>
							Remove
						</button>
					</li>
				{/each}
			</ul>
		{/if}

		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}

		<div class="actions">
			<button type="button" onclick={handleSubmit} disabled={selectedFiles.length === 0}>Submit</button>
			<button type="button" onclick={oncancel}>Cancel</button>
		</div>
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

	.queued-files {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: var(--space-1, 0.25rem);
	}

	.queued-files li {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2, 0.5rem);
		padding: var(--space-2, 0.5rem);
		background: var(--color-surface, #f8f9fa);
		border: 1px solid var(--color-border, #dee2e6);
		border-radius: var(--radius-sm, 0.25rem);
	}

	.queued-files li span {
		flex: 1 1 auto;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.actions {
		display: flex;
		gap: var(--space-2, 0.5rem);
	}
</style>
