<script lang="ts">
	interface Photo {
		key: string;
		url: string;
		thumbnail_url: string;
	}

	interface Props {
		photos: Photo[];
		removable?: boolean;
		onremove?: (key: string) => void;
		onphotoclick?: (url: string) => void;
	}

	let { photos, removable = false, onremove, onphotoclick }: Props = $props();
</script>

{#if photos.length > 0}
	<div class="photo-thumbnails">
		{#each photos as photo (photo.key)}
			<div class="thumbnail-wrapper">
				<button
					type="button"
					class="thumbnail-btn"
					onclick={() => onphotoclick?.(photo.url)}
				>
					<img
						src={photo.thumbnail_url || photo.url}
						alt=""
						class="thumbnail-img"
						loading="lazy"
					/>
				</button>
				{#if removable}
					<button
						type="button"
						class="remove-btn"
						aria-label="Remove photo"
						onclick={() => onremove?.(photo.key)}
					>&times;</button>
				{/if}
			</div>
		{/each}
	</div>
{/if}

<style>
	.photo-thumbnails {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}

	.thumbnail-wrapper {
		position: relative;
	}

	.thumbnail-btn {
		display: block;
		padding: 0;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		overflow: hidden;
		cursor: pointer;
		background: none;
		min-height: auto;
	}

	.thumbnail-img {
		display: block;
		width: 64px;
		height: 64px;
		object-fit: cover;
	}

	.remove-btn {
		position: absolute;
		top: -6px;
		right: -6px;
		width: 22px;
		height: 22px;
		min-height: auto;
		border-radius: 50%;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		color: var(--color-error);
		font-size: 14px;
		line-height: 1;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0;
	}
</style>
