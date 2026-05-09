<script lang="ts">
	import type { ImagingStudy } from '$lib/types/imaging';
	import { isPDFKey } from '$lib/types/imaging';
	import ImagingStudyViewer from './ImagingStudyViewer.svelte';

	interface Props {
		study: ImagingStudy;
	}

	let { study }: Props = $props();

	let photos = $derived(study.photos ?? []);

	let viewerOpen = $state(false);
	let viewerStartIndex = $state(0);

	function openViewer(idx: number) {
		viewerStartIndex = idx;
		viewerOpen = true;
	}

	function closeViewer() {
		viewerOpen = false;
	}
</script>

<section class="imaging-card">
	<header>
		<span class="badge">🖼️ {study.study_type}</span>
		<span class="date">{study.study_date}</span>
	</header>

	{#if study.notes}
		<p class="notes">{study.notes}</p>
	{/if}

	{#if photos.length > 0}
		<div class="thumbs">
			{#each photos as photo, i (photo.key)}
				<button
					type="button"
					class="thumb"
					onclick={() => openViewer(i)}
					aria-label="Open file {i + 1}"
				>
					{#if isPDFKey(photo.key) && !photo.thumbnail_url}
						<span class="pdf-icon">📄</span>
					{:else}
						<img src={photo.thumbnail_url || photo.url} alt="" />
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</section>

{#if viewerOpen}
	<ImagingStudyViewer
		photos={photos}
		startIndex={viewerStartIndex}
		onclose={closeViewer}
	/>
{/if}

<style>
	.imaging-card {
		background: var(--color-surface, #fff);
		border: 1px solid var(--color-border, #e0e0e0);
		border-left: 4px solid var(--color-primary, #0d6efd);
		border-radius: var(--radius-md, 6px);
		padding: var(--space-3, 1rem);
		box-shadow: var(--shadow-sm, 0 1px 2px rgba(0, 0, 0, 0.05));
	}
	header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		margin-bottom: var(--space-2, 0.5rem);
	}
	.badge {
		font-weight: 600;
		font-size: 0.875rem;
	}
	.date {
		color: var(--color-text-muted, #666);
		font-size: 0.875rem;
	}
	.notes {
		margin: var(--space-2, 0.5rem) 0;
		font-style: italic;
		color: var(--color-text, #333);
	}
	.thumbs {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}
	.thumb {
		padding: 0;
		border: 1px solid var(--color-border, #ccc);
		border-radius: 4px;
		background: white;
		cursor: pointer;
		width: 80px;
		height: 80px;
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
	}
	.thumb img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.pdf-icon {
		font-size: 32px;
	}
</style>
