<script lang="ts">
	import { apiClient } from '$lib/api';
	import { formatDateISO } from '$lib/datetime';
	import {
		IMAGING_QUICK_PICKS,
		isPDFKey,
		type ImagingExtractResponse
	} from '$lib/types/imaging';

	const MAX_FILES = 10;

	export interface ImagingStudyPayload {
		study_date: string;
		study_type: string;
		notes?: string;
		photo_keys: string[];
	}

	interface UploadState {
		key?: string;
		filename: string;
		isPDF: boolean;
		uploading: boolean;
		error?: string;
	}

	interface Props {
		babyId: string;
		onsubmit: (payload: ImagingStudyPayload) => void;
		submitting?: boolean;
		error?: string;
	}

	let { babyId, onsubmit, submitting = false, error = '' }: Props = $props();

	let studyDate = $state(formatDateISO(new Date()));
	let studyType = $state('');
	let notes = $state('');
	let uploads = $state<UploadState[]>([]);
	let toast = $state('');
	let extracting = $state(false);

	// Track which fields were auto-filled (so the user knows to verify).
	let autoFilledFields = $state<Set<string>>(new Set());

	let allUploadsDone = $derived(
		uploads.length > 0 && uploads.every((u) => !u.uploading && u.key)
	);

	let hasAnyUploads = $derived(uploads.length > 0);

	let canSubmit = $derived(
		!submitting &&
			allUploadsDone &&
			!extracting &&
			!!studyDate &&
			!!studyType.trim() &&
			!!uploads.find((u) => u.key)
	);

	async function uploadOne(file: File): Promise<string> {
		const fd = new FormData();
		fd.append('file', file);
		const resp = await apiClient.postForm<{ r2_key: string }>(
			`/babies/${babyId}/upload`,
			fd
		);
		return resp.r2_key;
	}

	async function handleFiles(event: Event) {
		const input = event.target as HTMLInputElement;
		if (!input.files || input.files.length === 0) return;

		const remaining = MAX_FILES - uploads.length;
		const files = Array.from(input.files).slice(0, remaining);

		if (input.files.length > remaining) {
			toast = `Only ${remaining} of ${input.files.length} files added (max ${MAX_FILES})`;
		}

		const newStates: UploadState[] = files.map((f) => ({
			filename: f.name,
			isPDF: f.type === 'application/pdf' || f.name.toLowerCase().endsWith('.pdf'),
			uploading: true
		}));
		const startIdx = uploads.length;
		uploads = [...uploads, ...newStates];

		// Upload each file, updating its own state on completion.
		await Promise.all(
			files.map(async (f, i) => {
				const idx = startIdx + i;
				try {
					const key = await uploadOne(f);
					uploads[idx] = { ...uploads[idx], key, uploading: false };
				} catch {
					uploads[idx] = {
						...uploads[idx],
						uploading: false,
						error: 'Upload failed'
					};
				}
				uploads = [...uploads];
			})
		);

		input.value = '';

		// After all uploads complete, run auto-extract if at least one succeeded.
		if (uploads.some((u) => u.key)) {
			void runExtract();
		}
	}

	function removeUpload(i: number) {
		uploads = uploads.filter((_, idx) => idx !== i);
	}

	async function runExtract() {
		const keys = uploads.filter((u) => u.key).map((u) => u.key as string);
		if (keys.length === 0) return;
		extracting = true;
		try {
			const resp = await apiClient.post<ImagingExtractResponse>(
				`/babies/${babyId}/imaging-studies/extract`,
				{ photo_keys: keys }
			);
			applySuggestion(resp);
		} catch {
			toast = "Couldn't analyze, please fill manually";
		} finally {
			extracting = false;
		}
	}

	function applySuggestion(resp: ImagingExtractResponse) {
		const s = resp.suggested;
		// User-typed values always win: only set when current value is empty
		// or already auto-filled. Don't overwrite user edits.
		if (s.study_type && (!studyType || autoFilledFields.has('study_type'))) {
			studyType = s.study_type;
			autoFilledFields.add('study_type');
		}
		if (
			s.study_date &&
			(studyDate === formatDateISO(new Date()) || autoFilledFields.has('study_date'))
		) {
			studyDate = s.study_date;
			autoFilledFields.add('study_date');
		}
		const findingsText = s.findings || s.notes || '';
		if (findingsText && (!notes || autoFilledFields.has('notes'))) {
			notes = findingsText;
			autoFilledFields.add('notes');
		}
		autoFilledFields = new Set(autoFilledFields);
	}

	function selectQuickPick(pick: string) {
		studyType = pick;
		autoFilledFields.delete('study_type');
		autoFilledFields = new Set(autoFilledFields);
	}

	function onTypeInput() {
		autoFilledFields.delete('study_type');
		autoFilledFields = new Set(autoFilledFields);
	}

	function onDateInput() {
		autoFilledFields.delete('study_date');
		autoFilledFields = new Set(autoFilledFields);
	}

	function onNotesInput() {
		autoFilledFields.delete('notes');
		autoFilledFields = new Set(autoFilledFields);
	}

	function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		const keys = uploads.filter((u) => u.key).map((u) => u.key as string);
		if (keys.length === 0) {
			toast = 'Add at least one image or PDF';
			return;
		}
		const payload: ImagingStudyPayload = {
			study_date: studyDate,
			study_type: studyType.trim(),
			photo_keys: keys
		};
		if (notes.trim()) payload.notes = notes.trim();
		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit} class="imaging-form">
	<div class="files">
		<label for="imaging-files" class="file-label">Add images / PDFs (max {MAX_FILES})</label>
		<input
			id="imaging-files"
			type="file"
			accept="image/*,application/pdf"
			multiple
			onchange={handleFiles}
			disabled={submitting}
		/>
		{#if hasAnyUploads}
			<ul class="upload-list">
				{#each uploads as u, i (u.filename + i)}
					<li class:err={u.error}>
						<span>{u.isPDF ? '📄' : '🖼️'} {u.filename}</span>
						{#if u.uploading}
							<span class="status">Uploading…</span>
						{:else if u.error}
							<span class="status">{u.error}</span>
						{:else}
							<span class="status">✓</span>
						{/if}
						<button type="button" onclick={() => removeUpload(i)} class="remove">
							Remove
						</button>
					</li>
				{/each}
			</ul>
		{/if}
		{#if extracting}
			<p class="hint">Analyzing…</p>
		{/if}
	</div>

	<div class="quick-picks">
		<span>Quick pick:</span>
		{#each IMAGING_QUICK_PICKS as pick (pick)}
			<button
				type="button"
				class:selected={studyType === pick}
				onclick={() => selectQuickPick(pick)}
			>
				{pick}
			</button>
		{/each}
	</div>

	<div>
		<label for="imaging-type">
			Study type
			{#if autoFilledFields.has('study_type')}
				<span class="auto-fill" title="auto-filled — please verify">✨</span>
			{/if}
		</label>
		<input
			id="imaging-type"
			type="text"
			bind:value={studyType}
			oninput={onTypeInput}
			class:auto={autoFilledFields.has('study_type')}
			placeholder="CT, Ultrasound, MRI, HIDA, …"
		/>
	</div>

	<div>
		<label for="imaging-date">
			Study date
			{#if autoFilledFields.has('study_date')}
				<span class="auto-fill" title="auto-filled — please verify">✨</span>
			{/if}
		</label>
		<input
			id="imaging-date"
			type="date"
			bind:value={studyDate}
			oninput={onDateInput}
			class:auto={autoFilledFields.has('study_date')}
		/>
	</div>

	<div>
		<label for="imaging-notes">
			Notes
			{#if autoFilledFields.has('notes')}
				<span class="auto-fill" title="auto-filled — please verify">✨</span>
			{/if}
		</label>
		<textarea
			id="imaging-notes"
			bind:value={notes}
			oninput={onNotesInput}
			class:auto={autoFilledFields.has('notes')}
			rows="4"
		></textarea>
	</div>

	{#if toast}
		<p role="alert" class="toast">{toast}</p>
	{/if}
	{#if error}
		<p role="alert" class="error">{error}</p>
	{/if}

	<button type="submit" disabled={!canSubmit}>
		{submitting ? 'Saving…' : 'Save imaging study'}
	</button>
</form>

<style>
	.imaging-form {
		display: flex;
		flex-direction: column;
		gap: var(--space-3, 1rem);
	}
	.files .file-label {
		display: block;
		margin-bottom: var(--space-2, 0.5rem);
		font-weight: 600;
	}
	.upload-list {
		list-style: none;
		padding: 0;
		margin: var(--space-2, 0.5rem) 0 0;
	}
	.upload-list li {
		display: flex;
		gap: var(--space-2, 0.5rem);
		align-items: center;
		padding: 4px 0;
		border-bottom: 1px solid var(--color-border, #e0e0e0);
	}
	.upload-list li.err {
		color: var(--color-error, #b91c1c);
	}
	.upload-list .status {
		color: var(--color-text-muted, #666);
		font-size: 0.875rem;
	}
	.upload-list .remove {
		margin-left: auto;
		background: none;
		border: 1px solid var(--color-border, #ccc);
		color: var(--color-text-muted, #666);
		font-size: 0.75rem;
		padding: 2px 8px;
		min-height: auto;
		cursor: pointer;
	}
	.quick-picks {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
		align-items: center;
	}
	.quick-picks button {
		padding: var(--space-1, 0.25rem) var(--space-3, 1rem);
	}
	.quick-picks button.selected {
		background: var(--color-primary, #0d6efd);
		color: white;
	}
	.auto-fill {
		display: inline-block;
		margin-left: 4px;
	}
	.auto {
		background-color: var(--color-warning-bg, #fff8e1);
	}
	.toast {
		color: var(--color-warning, #92400e);
		background: var(--color-warning-bg, #fef3c7);
		padding: var(--space-2, 0.5rem);
		border-radius: var(--radius-sm, 4px);
	}
	.error {
		color: var(--color-error, #b91c1c);
	}
	.hint {
		color: var(--color-text-muted, #666);
		font-size: 0.875rem;
		margin: 0;
	}
</style>
