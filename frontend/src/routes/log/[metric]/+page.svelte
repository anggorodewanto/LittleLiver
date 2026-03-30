<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { activeBaby } from '$lib/stores/baby';
	import { apiClient } from '$lib/api';

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

	// Medication edit mode
	let editMedicationId = $derived($page.url.searchParams.get('edit') ?? '');
	let editMedicationData = $state<import('$lib/components/MedicationForm.svelte').MedicationInitialData | undefined>(undefined);
	let loadingEdit = $state(false);

	$effect(() => {
		if (metric === 'medication' && editMedicationId && baby) {
			loadingEdit = true;
			apiClient.get<{ id: string; name: string; dose: string; frequency: string; schedule: string | null; active: boolean }>(`/babies/${baby.id}/medications/${editMedicationId}`)
				.then((med) => {
					let scheduleTimes: string[] = [];
					if (med.schedule) {
						try { scheduleTimes = JSON.parse(med.schedule); } catch { /* empty */ }
					}
					editMedicationData = {
						name: med.name,
						dose: med.dose,
						frequency: med.frequency,
						schedule_times: scheduleTimes,
						active: med.active
					};
				})
				.catch(() => { error = 'Failed to load medication'; })
				.finally(() => { loadingEdit = false; });
		} else {
			editMedicationData = undefined;
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
			// Medication edit mode: use PUT instead of POST
			if (metric === 'medication' && editMedicationId) {
				await apiClient.put(`/babies/${baby.id}/medications/${editMedicationId}`, data);
				goto('/medications');
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

<a href="/" class="back-link">&larr; Back</a>

{#if !baby}
	<p>No baby selected</p>
{:else if !config}
	<p>Unknown metric type</p>
{:else}
	<h1>{editMedicationId ? 'Edit' : 'Log'} {config.label}</h1>

	{#if metric === 'feeding'}
		<FeedingForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'urine'}
		<UrineForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'stool'}
		<StoolForm onsubmit={handleSubmit} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'temperature'}
		<TemperatureForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'weight'}
		<WeightForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'abdomen'}
		<AbdomenForm onsubmit={handleSubmit} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'skin'}
		<SkinForm onsubmit={handleSubmit} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'bruising'}
		<BruisingForm onsubmit={handleSubmit} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'lab'}
		<LabForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'notes'}
		<NotesForm onsubmit={handleSubmit} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'medication'}
		{#if loadingEdit}
			<div class="loading">Loading medication...</div>
		{:else}
			<MedicationForm onsubmit={handleSubmit} initialData={editMedicationData} {submitting} {error} />
		{/if}
	{:else if metric === 'med'}
		<DoseLogForm onsubmit={handleSubmit} babyId={baby.id} {medicationId} {scheduledTime} {submitting} {error} />
	{:else if metric === 'other_intake'}
		<FluidLogForm direction="intake" onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'other_output'}
		<FluidLogForm direction="output" onsubmit={handleSubmit} {submitting} {error} />
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
