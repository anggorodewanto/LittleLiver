<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { activeBaby } from '$lib/stores/baby';
	import { apiClient } from '$lib/api';

	import { fromISO8601 } from '$lib/datetime';
	import FeedingForm from '$lib/components/FeedingForm.svelte';
	import UrineForm from '$lib/components/UrineForm.svelte';
	import StoolForm from '$lib/components/StoolForm.svelte';
	import TemperatureForm from '$lib/components/TemperatureForm.svelte';
	import WeightForm from '$lib/components/WeightForm.svelte';
	import AbdomenForm from '$lib/components/AbdomenForm.svelte';
	import SkinForm from '$lib/components/SkinForm.svelte';
	import BruisingForm from '$lib/components/BruisingForm.svelte';
	import LabForm from '$lib/components/LabForm.svelte';
	import NotesForm from '$lib/components/NotesForm.svelte';
	import MedicationForm from '$lib/components/MedicationForm.svelte';
	import DoseLogForm from '$lib/components/DoseLogForm.svelte';
	import FluidLogForm from '$lib/components/FluidLogForm.svelte';
	import HeadCircumferenceForm from '$lib/components/HeadCircumferenceForm.svelte';
	import UpperArmCircumferenceForm from '$lib/components/UpperArmCircumferenceForm.svelte';
	import LabImportFlow from '$lib/components/LabImportFlow.svelte';

	const METRIC_CONFIG: Record<string, { label: string; endpoint: string; hasPhoto: boolean; multiPhoto?: boolean }> = {
		feeding: { label: 'Feeding', endpoint: 'feedings', hasPhoto: false },
		urine: { label: 'Urine', endpoint: 'urine', hasPhoto: false },
		stool: { label: 'Stool', endpoint: 'stools', hasPhoto: true, multiPhoto: true },
		temperature: { label: 'Temperature', endpoint: 'temperatures', hasPhoto: false },
		weight: { label: 'Weight', endpoint: 'weights', hasPhoto: false },
		abdomen: { label: 'Abdomen', endpoint: 'abdomen', hasPhoto: true, multiPhoto: true },
		skin: { label: 'Skin', endpoint: 'skin', hasPhoto: true, multiPhoto: true },
		bruising: { label: 'Bruising', endpoint: 'bruising', hasPhoto: true, multiPhoto: true },
		lab: { label: 'Lab', endpoint: 'labs', hasPhoto: false },
		notes: { label: 'Note', endpoint: 'notes', hasPhoto: true, multiPhoto: true },
		medication: { label: 'Medication', endpoint: 'medications', hasPhoto: false },
		med: { label: 'Dose', endpoint: 'med-logs', hasPhoto: false },
		head_circumference: { label: 'Head Circumference', endpoint: 'head-circumferences', hasPhoto: false },
		upper_arm_circumference: { label: 'Upper Arm Circumference', endpoint: 'upper-arm-circumferences', hasPhoto: false },
		other_intake: { label: 'Other Intake', endpoint: 'fluid-log', hasPhoto: false },
		other_output: { label: 'Other Output', endpoint: 'fluid-log', hasPhoto: false }
	};

	let metric = $derived($page.params.metric);
	let baby = $derived($activeBaby);
	let config = $derived(METRIC_CONFIG[metric]);

	interface PhotoInfo {
		key: string;
		url: string;
		thumbnail_url: string;
	}

	let submitting = $state(false);
	let error = $state('');
	let uploading = $state(false);
	let photoKeys = $state<string[]>([]);
	let existingPhotos = $state<PhotoInfo[]>([]);
	let showLabImport = $state(false);

	$effect(() => {
		void metric;  // track the metric reactive dependency
		photoKeys = [];
		existingPhotos = [];
		error = '';
		submitting = false;
		uploading = false;
	});

	// Query params for DoseLogForm
	let medicationId = $derived($page.url.searchParams.get('medicationId') ?? '');
	let scheduledTime = $derived($page.url.searchParams.get('scheduled_time') ?? undefined);

	// Generic edit mode
	let editId = $derived($page.url.searchParams.get('edit') ?? '');
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let editData = $state<any>(undefined);
	let loadingEdit = $state(false);

	function transformForEdit(metricKey: string, raw: Record<string, unknown>): unknown {
		if (metricKey === 'medication') {
			return {
				name: raw.name,
				dose: raw.dose,
				frequency: raw.frequency,
				schedule_times: (raw.schedule_times as string[] | null) ?? [],
				active: raw.active,
				interval_days: raw.interval_days as number | undefined,
				starts_from: raw.starts_from as string | undefined,
				notes: raw.notes as string | undefined
			};
		}
		if (metricKey === 'med') {
			return { medication_id: raw.medication_id, skipped: raw.skipped, given_at: raw.given_at, skip_reason: raw.skip_reason, notes: raw.notes };
		}
		// All other types have a timestamp field — return as-is (forms use fromISO8601 internally)
		return raw;
	}

	$effect(() => {
		if (editId && baby && config) {
			loadingEdit = true;
			editData = undefined;
			apiClient.get<Record<string, unknown>>(`/babies/${baby.id}/${config.endpoint}/${editId}`)
				.then((raw) => {
					editData = transformForEdit(metric, raw);
					if (Array.isArray(raw.photos) && raw.photos.length > 0) {
						const photos = raw.photos as PhotoInfo[];
						photoKeys = photos.map(p => p.key);
						existingPhotos = photos;
					}
				})
				.catch(() => { error = 'Failed to load entry'; })
				.finally(() => { loadingEdit = false; });
		} else {
			editData = undefined;
		}
	});

	async function uploadPhoto(babyId: string, file: File): Promise<string> {
		const formData = new FormData();
		formData.append('file', file);
		const data = await apiClient.postForm<{ r2_key: string }>(`/babies/${babyId}/upload`, formData);
		return data.r2_key;
	}

	function handlePhotoRemove(key: string): void {
		photoKeys = photoKeys.filter(k => k !== key);
		existingPhotos = existingPhotos.filter(p => p.key !== key);
	}

	async function handlePhotoUpload(file: File): Promise<void> {
		if (!baby) return;
		uploading = true;
		try {
			const key = await uploadPhoto(baby.id, file);
			photoKeys = [...photoKeys, key];
		} catch {
			error = 'Photo upload failed';
		} finally {
			uploading = false;
		}
	}

	async function handleSubmit(data: unknown): Promise<void> {
		if (!baby || !config) return;
		submitting = true;
		error = '';
		try {
			if (editId) {
				await apiClient.put(`/babies/${baby.id}/${config.endpoint}/${editId}`, data);
				if (metric === 'medication') {
					goto('/medications');
				} else {
					goto('/logs');
				}
			} else if (Array.isArray(data)) {
				for (const entry of data) {
					await apiClient.post(`/babies/${baby.id}/${config.endpoint}`, entry);
				}
				goto('/');
			} else {
				await apiClient.post(`/babies/${baby.id}/${config.endpoint}`, data);
				goto('/');
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			submitting = false;
		}
	}
</script>

<a href={editId ? (metric === 'medication' ? '/medications' : '/logs') : '/'} class="back-link">&larr; Back</a>

{#if !baby}
	<p>No baby selected</p>
{:else if !config}
	<p>Unknown metric type</p>
{:else if loadingEdit}
	<div class="loading">Loading...</div>
{:else}
	<h1>{editId ? 'Edit' : 'Log'} {config.label}</h1>

	{#if metric === 'feeding'}
		<FeedingForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'urine'}
		<UrineForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'stool'}
		<StoolForm onsubmit={handleSubmit} initialData={editData} onphotoupload={handlePhotoUpload} onphotoremove={handlePhotoRemove} {submitting} {error} {uploading} {photoKeys} {existingPhotos} />
	{:else if metric === 'temperature'}
		<TemperatureForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'weight'}
		<WeightForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'abdomen'}
		<AbdomenForm onsubmit={handleSubmit} initialData={editData} onphotoupload={handlePhotoUpload} onphotoremove={handlePhotoRemove} {submitting} {error} {uploading} {photoKeys} {existingPhotos} />
	{:else if metric === 'skin'}
		<SkinForm onsubmit={handleSubmit} initialData={editData} onphotoupload={handlePhotoUpload} onphotoremove={handlePhotoRemove} {submitting} {error} {uploading} {photoKeys} {existingPhotos} />
	{:else if metric === 'bruising'}
		<BruisingForm onsubmit={handleSubmit} initialData={editData} onphotoupload={handlePhotoUpload} onphotoremove={handlePhotoRemove} {submitting} {error} {uploading} {photoKeys} {existingPhotos} />
	{:else if metric === 'lab'}
		{#if showLabImport}
			<LabImportFlow babyId={baby.id} oncancel={() => { showLabImport = false; }} onsaved={() => { goto('/'); }} />
		{:else}
			{#if !editId}
				<button type="button" class="import-photo-btn" onclick={() => { showLabImport = true; }}>
					Import from photo
				</button>
			{/if}
			<LabForm onsubmit={handleSubmit} initialData={editData} babyId={baby.id} {submitting} {error} />
		{/if}
	{:else if metric === 'notes'}
		<NotesForm onsubmit={handleSubmit} initialData={editData} onphotoupload={handlePhotoUpload} onphotoremove={handlePhotoRemove} {submitting} {error} {uploading} {photoKeys} {existingPhotos} />
	{:else if metric === 'medication'}
		<MedicationForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'med'}
		<DoseLogForm onsubmit={handleSubmit} initialData={editData} babyId={baby.id} {medicationId} {scheduledTime} {submitting} {error} />
	{:else if metric === 'head_circumference'}
		<HeadCircumferenceForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'upper_arm_circumference'}
		<UpperArmCircumferenceForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'other_intake'}
		<FluidLogForm direction="intake" onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'other_output'}
		<FluidLogForm direction="output" onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{/if}
{/if}

<style>
	.back-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		margin-bottom: var(--space-3);
		min-height: var(--touch-target);
		text-decoration: none;
	}

	.back-link:hover {
		color: var(--color-primary);
	}

	.import-photo-btn {
		width: 100%;
		margin-bottom: var(--space-3, 1rem);
		background: var(--color-surface, #f8f9fa);
		border: 2px dashed var(--color-border, #e0e0e0);
		border-radius: var(--radius, 8px);
		padding: var(--space-2, 0.5rem) var(--space-3, 1rem);
		cursor: pointer;
		font-weight: 500;
	}
</style>
