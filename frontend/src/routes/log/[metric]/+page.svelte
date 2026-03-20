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

	const METRIC_CONFIG: Record<string, { label: string; endpoint: string; hasPhoto: boolean; multiPhoto?: boolean }> = {
		feeding: { label: 'Feeding', endpoint: 'feedings', hasPhoto: false },
		urine: { label: 'Urine', endpoint: 'urine', hasPhoto: false },
		stool: { label: 'Stool', endpoint: 'stools', hasPhoto: true },
		temperature: { label: 'Temperature', endpoint: 'temperatures', hasPhoto: false },
		weight: { label: 'Weight', endpoint: 'weights', hasPhoto: false },
		abdomen: { label: 'Abdomen', endpoint: 'abdomen', hasPhoto: true },
		skin: { label: 'Skin', endpoint: 'skin', hasPhoto: true },
		bruising: { label: 'Bruising', endpoint: 'bruising', hasPhoto: true },
		lab: { label: 'Lab', endpoint: 'labs', hasPhoto: false },
		notes: { label: 'Note', endpoint: 'notes', hasPhoto: true, multiPhoto: true },
		medication: { label: 'Medication', endpoint: 'medications', hasPhoto: false },
		med: { label: 'Dose', endpoint: 'med-logs', hasPhoto: false }
	};

	let metric = $derived($page.params.metric);
	let baby = $derived($activeBaby);
	let config = $derived(METRIC_CONFIG[metric]);

	let submitting = $state(false);
	let error = $state('');
	let uploading = $state(false);
	let photoKey = $state('');
	let photoKeys = $state<string[]>([]);

	// Query params for DoseLogForm
	let medicationId = $derived($page.url.searchParams.get('medication_id') ?? '');
	let scheduledTime = $derived($page.url.searchParams.get('scheduled_time') ?? undefined);

	async function uploadPhoto(babyId: string, file: File): Promise<string> {
		const csrfRes = await fetch('/api/csrf-token', { credentials: 'include' });
		const { csrf_token } = await csrfRes.json();

		const formData = new FormData();
		formData.append('photo', file);
		const res = await fetch(`/api/babies/${babyId}/upload`, {
			method: 'POST',
			body: formData,
			credentials: 'include',
			headers: {
				'X-CSRF-Token': csrf_token,
				'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone
			}
		});

		if (!res.ok) throw new Error('Upload failed');
		const data = await res.json();
		return data.r2_key;
	}

	async function handlePhotoUpload(file: File): Promise<void> {
		if (!baby) return;
		uploading = true;
		try {
			const key = await uploadPhoto(baby.id, file);
			if (config?.multiPhoto) {
				photoKeys = [...photoKeys, key];
			} else {
				photoKey = key;
			}
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
			await apiClient.post(`/babies/${baby.id}/${config.endpoint}`, data);
			goto('/');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			submitting = false;
		}
	}
</script>

<a href="/">Back</a>

{#if !baby}
	<p>No baby selected</p>
{:else if !config}
	<p>Unknown metric type</p>
{:else}
	<h1>Log {config.label}</h1>

	{#if metric === 'feeding'}
		<FeedingForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'urine'}
		<UrineForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'stool'}
		<StoolForm onsubmit={handleSubmit} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKey} />
	{:else if metric === 'temperature'}
		<TemperatureForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'weight'}
		<WeightForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'abdomen'}
		<AbdomenForm onsubmit={handleSubmit} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKey} />
	{:else if metric === 'skin'}
		<SkinForm onsubmit={handleSubmit} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKey} />
	{:else if metric === 'bruising'}
		<BruisingForm onsubmit={handleSubmit} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKey} />
	{:else if metric === 'lab'}
		<LabForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'notes'}
		<NotesForm onsubmit={handleSubmit} onphotoupload={handlePhotoUpload} {submitting} {error} {uploading} {photoKeys} />
	{:else if metric === 'medication'}
		<MedicationForm onsubmit={handleSubmit} {submitting} {error} />
	{:else if metric === 'med'}
		<DoseLogForm onsubmit={handleSubmit} babyId={baby.id} {medicationId} {scheduledTime} {submitting} {error} />
	{/if}
{/if}
