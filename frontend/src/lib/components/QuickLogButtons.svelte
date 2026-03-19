<script lang="ts">
	interface Props {
		onselect: (type: string) => void;
	}

	let { onselect }: Props = $props();
</script>

<div class="quick-log-buttons">
	<button type="button" onclick={() => onselect('feeding')}>Feed</button>
	<button type="button" onclick={() => onselect('urine')}>Wet Diaper</button>
	<button type="button" onclick={() => onselect('stool')}>Stool</button>
	<button type="button" onclick={() => onselect('temperature')}>Temp</button>
	<button type="button" onclick={() => onselect('medication')}>Medication</button>
</div>
