<script lang="ts">
	export type MetricType = 'feeding' | 'urine' | 'stool' | 'temperature' | 'med_given' | 'weight' | 'abdomen' | 'skin' | 'bruising' | 'lab' | 'notes' | 'medication';

	interface Props {
		onselect: (type: MetricType) => void;
		onnavigate?: (path: string) => void;
	}

	let { onselect, onnavigate }: Props = $props();

	let expanded = $state(false);
</script>

<div class="quick-log-buttons">
	<button type="button" onclick={() => onselect('feeding')}>Feed</button>
	<button type="button" onclick={() => onselect('urine')}>Wet Diaper</button>
	<button type="button" onclick={() => onselect('stool')}>Stool</button>
	<button type="button" onclick={() => onselect('temperature')}>Temp</button>
	<button type="button" onclick={() => onselect('med_given')}>Medication Given</button>
</div>

<div class="more-entries">
	<button type="button" class="toggle-more" onclick={() => expanded = !expanded}>
		{expanded ? 'Less Entries' : 'More Entries'}
	</button>

	{#if expanded}
		<div class="extra-buttons">
			<button type="button" onclick={() => onselect('weight')}>Weight</button>
			<button type="button" onclick={() => onselect('abdomen')}>Abdomen</button>
			<button type="button" onclick={() => onselect('skin')}>Skin</button>
			<button type="button" onclick={() => onselect('bruising')}>Bruising</button>
			<button type="button" onclick={() => onselect('lab')}>Lab</button>
			<button type="button" onclick={() => onselect('notes')}>Notes</button>
			<button type="button" onclick={() => onnavigate?.('/medications')}>Manage Medications</button>
		</div>
	{/if}
</div>
