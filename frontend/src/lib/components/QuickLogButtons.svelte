<script lang="ts">
	export type MetricType = 'feeding' | 'urine' | 'stool' | 'temperature' | 'med_given' | 'weight' | 'abdomen' | 'skin' | 'bruising' | 'lab' | 'notes' | 'medication' | 'other_intake' | 'other_output' | 'head_circumference' | 'upper_arm_circumference';

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
			<button type="button" onclick={() => onselect('head_circumference')}>Head Circ.</button>
			<button type="button" onclick={() => onselect('upper_arm_circumference')}>Arm Circ.</button>
			<button type="button" onclick={() => onselect('other_intake')}>Other Intake</button>
			<button type="button" onclick={() => onselect('other_output')}>Other Output</button>
			<button type="button" onclick={() => onnavigate?.('/medications')}>Manage Medications</button>
		</div>
	{/if}
</div>

<style>
	.quick-log-buttons {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: var(--space-2);
		margin-bottom: var(--space-3);
	}

	.quick-log-buttons button {
		min-height: 52px;
		font-weight: 600;
		font-size: var(--font-size-sm);
		background: var(--color-primary);
		color: var(--color-text-inverse);
		border-radius: var(--radius-md);
	}

	.quick-log-buttons button:hover {
		background: var(--color-primary-dark);
	}

	.more-entries {
		text-align: center;
	}

	.toggle-more {
		font-size: var(--font-size-sm);
		background: transparent;
		color: var(--color-text-muted);
		border: 1px dashed var(--color-border);
		width: 100%;
		border-radius: var(--radius-sm);
	}

	.toggle-more:hover {
		background: var(--color-primary-light);
		color: var(--color-primary-dark);
	}

	.extra-buttons {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: var(--space-2);
		margin-top: var(--space-2);
	}

	.extra-buttons button {
		min-height: 48px;
		font-size: var(--font-size-sm);
		background: var(--color-surface);
		color: var(--color-text);
		border: 1.5px solid var(--color-border);
		border-radius: var(--radius-md);
	}

	.extra-buttons button:hover {
		border-color: var(--color-primary);
		background: var(--color-primary-light);
	}
</style>
