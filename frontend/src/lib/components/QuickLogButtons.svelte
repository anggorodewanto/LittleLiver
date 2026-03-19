<script lang="ts">
	export type MetricType = 'feeding' | 'urine' | 'stool' | 'temperature' | 'med_given';

	interface Props {
		onselect: (type: MetricType) => void;
	}

	let { onselect }: Props = $props();
</script>

<div class="quick-log-buttons">
	<button type="button" onclick={() => onselect('feeding')}>Feed</button>
	<button type="button" onclick={() => onselect('urine')}>Wet Diaper</button>
	<button type="button" onclick={() => onselect('stool')}>Stool</button>
	<button type="button" onclick={() => onselect('temperature')}>Temp</button>
	<button type="button" onclick={() => onselect('med_given')}>Medication Given</button>
</div>
