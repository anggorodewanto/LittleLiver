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
		other_intake: { label: 'Other Intake', endpoint: 'fluid-log', hasPhoto: false },
		other_output: { label: 'Other Output', endpoint: 'fluid-log', hasPhoto: false }
	};

	let metric = $derived($page.params.metric);
	let baby = $derived($activeBaby);
	let config = $derived(METRIC_CONFIG[metric]);

	let submitting = $state(false);
	let error = $state('');
	let uploading = $state(false);
	let photoKeys = $state<string[]>([]);

	$effect(() => {
		void metric;  // track the metric reactive dependency
		photoKeys = [];
		error = '';
		submitting = false;
		uploading = false;
	});

	// Query params for DoseLogForm
	let medicationId = $derived($page.url.searchParams.get('medication_id') ?? '');
	let scheduledTime = $derived($page.url.searchParams.get('scheduled_time') ?? undefined);

	// Generic edit mode
	let editId = $derived($page.url.searchParams.get('edit') ?? '');
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let editData = $state<any>(undefined);
	let loadingEdit = $state(false);

	function transformForEdit(metricKey: string, raw: Record<string, unknown>): unknown {
		if (metricKey === 'medication') {
			const schedule = raw.schedule as string | null;
			let scheduleTimes: string[] = [];
			if (schedule) {
				try { scheduleTimes = JSON.parse(schedule); } catch { /* empty */ }
			}
			return { name: raw.name, dose: raw.dose, frequency: raw.frequency, schedule_times: scheduleTimes, active: raw.active, interval_days: raw.interval_days as number | undefined };
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
				.then((raw) => { editData = transformForEdit(metric, raw); })
				.catch(() => { error = 'Failed to load entry'; })
				.finally(() => { loadingEdit = false; });
		} else {
			editData = undefined;
		}
	});

	async function uploadPhoto(babyId: string, file: File): Promise<string> {
		const formData = new FormData();
		formData.append('photo', file);
		const data = await apiClient.postForm<{ r2_key: string }>(`/babies/${babyId}/upload`, formData);
		return data.r2_key;
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

<a href={editId ? '/logs' : '/'} class="back-link">&larr; Back</a>

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
		<StoolForm onsubmit={handleSubmit} initialData={editData} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'temperature'}
		<TemperatureForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'weight'}
		<WeightForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'abdomen'}
		<AbdomenForm onsubmit={handleSubmit} initialData={editData} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'skin'}
		<SkinForm onsubmit={handleSubmit} initialData={editData} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'bruising'}
		<BruisingForm onsubmit={handleSubmit} initialData={editData} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'lab'}
		<LabForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'notes'}
		<NotesForm onsubmit={handleSubmit} initialData={editData} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'medication'}
		<MedicationForm onsubmit={handleSubmit} initialData={editData} {submitting} {error} />
	{:else if metric === 'med'}
		<DoseLogForm onsubmit={handleSubmit} initialData={editData} babyId={baby.id} {medicationId} {scheduledTime} {submitting} {error} />
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
</style>
