<script lang="ts" module>
	export type PhaseUnit = 'days' | 'weeks' | 'months';

	export interface PhaseDraft {
		seq: number;
		label: string;
		start_date: string;
	}

	export interface CarePlanFormPayload {
		name: string;
		notes?: string;
		phases: PhaseDraft[];
	}

	export interface CarePlanFormInitial {
		name: string;
		notes?: string;
		phases: PhaseDraft[];
	}

	function pad2(n: number): string {
		return n < 10 ? `0${n}` : `${n}`;
	}

	function toDateStr(d: Date): string {
		return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
	}

	export function generatePhases(
		count: number,
		unit: PhaseUnit,
		startDate: string,
		labelTemplate: string
	): PhaseDraft[] {
		const start = new Date(`${startDate}T00:00:00`);
		if (Number.isNaN(start.getTime()) || count < 1) {
			return [];
		}
		const phases: PhaseDraft[] = [];
		for (let i = 0; i < count; i++) {
			const d = new Date(start.getTime());
			if (unit === 'days') d.setDate(d.getDate() + i);
			if (unit === 'weeks') d.setDate(d.getDate() + i * 7);
			if (unit === 'months') d.setMonth(d.getMonth() + i);
			phases.push({
				seq: i + 1,
				label: labelTemplate.replace('{n}', String(i + 1)),
				start_date: toDateStr(d)
			});
		}
		return phases;
	}

	export function validatePhases(phases: PhaseDraft[]): string {
		if (phases.length === 0) return 'At least one phase is required';
		for (let i = 0; i < phases.length; i++) {
			const p = phases[i];
			if (p.seq !== i + 1) return `Phase seq must be ${i + 1}`;
			if (!p.label.trim()) return `Phase ${i + 1} label is required`;
			if (!p.start_date) return `Phase ${i + 1} start date is required`;
			if (i > 0 && p.start_date <= phases[i - 1].start_date) {
				return `Phase ${i + 1} start date must be after phase ${i}`;
			}
		}
		return '';
	}
</script>

<script lang="ts">
	interface Props {
		onsubmit: (data: CarePlanFormPayload) => void;
		initialData?: CarePlanFormInitial;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	let name = $state('');
	let notes = $state('');
	let phases = $state<PhaseDraft[]>([]);
	let validationError = $state('');

	$effect(() => {
		name = initialData?.name ?? '';
		notes = initialData?.notes ?? '';
		phases = initialData?.phases ?? [];
		validationError = '';
	});

	let genCount = $state(12);
	let genUnit = $state<PhaseUnit>('months');
	let genStart = $state('');
	let genTemplate = $state('Phase {n}');

	function runGenerator(): void {
		const generated = generatePhases(genCount, genUnit, genStart, genTemplate);
		if (generated.length === 0) {
			validationError = 'Generator needs a valid start date and count >= 1';
			return;
		}
		if (phases.length > 0 && !confirm('Replace existing phases?')) {
			return;
		}
		phases = generated;
		validationError = '';
	}

	function addPhase(): void {
		const last = phases[phases.length - 1];
		const next: PhaseDraft = {
			seq: phases.length + 1,
			label: '',
			start_date: last?.start_date ?? ''
		};
		phases = [...phases, next];
	}

	function removePhase(idx: number): void {
		phases = phases.filter((_, i) => i !== idx).map((p, i) => ({ ...p, seq: i + 1 }));
	}

	function updateLabel(idx: number, value: string): void {
		phases = phases.map((p, i) => (i === idx ? { ...p, label: value } : p));
	}

	function updateStart(idx: number, value: string): void {
		phases = phases.map((p, i) => (i === idx ? { ...p, start_date: value } : p));
		const msg = validatePhases(phases);
		validationError = msg;
	}

	function handleSubmit(event: SubmitEvent): void {
		event.preventDefault();
		if (!name.trim()) {
			validationError = 'Name is required';
			return;
		}
		const msg = validatePhases(phases);
		if (msg) {
			validationError = msg;
			return;
		}
		validationError = '';
		onsubmit({
			name: name.trim(),
			notes: notes.trim() || undefined,
			phases
		});
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="cp-name">Plan Name</label>
		<input id="cp-name" type="text" bind:value={name} placeholder="e.g. Antibiotic Rotation" />
	</div>

	<div>
		<label for="cp-notes">Notes</label>
		<textarea id="cp-notes" bind:value={notes}></textarea>
	</div>

	<fieldset class="generator">
		<legend>Generate phases</legend>
		<div>
			<label for="gen-count">Count</label>
			<input id="gen-count" type="number" min="1" bind:value={genCount} />
		</div>
		<div>
			<label for="gen-unit">Each lasts</label>
			<select id="gen-unit" bind:value={genUnit}>
				<option value="days">Days</option>
				<option value="weeks">Weeks</option>
				<option value="months">Months</option>
			</select>
		</div>
		<div>
			<label for="gen-start">Start date</label>
			<input id="gen-start" type="date" bind:value={genStart} />
		</div>
		<div>
			<label for="gen-template">Label template</label>
			<input id="gen-template" type="text" bind:value={genTemplate} />
			<small>Use {'{n}'} for the phase number.</small>
		</div>
		<button type="button" onclick={runGenerator}>Generate</button>
	</fieldset>

	<h3>Phases</h3>
	{#if phases.length === 0}
		<p>No phases yet. Use the generator or add manually.</p>
	{:else}
		<table>
			<thead>
				<tr><th>#</th><th>Label</th><th>Start date</th><th></th></tr>
			</thead>
			<tbody>
				{#each phases as phase, i (i)}
					<tr>
						<td>{phase.seq}</td>
						<td>
							<label for="phase-label-{i}" class="sr-only">Phase {i + 1} label</label>
							<input
								id="phase-label-{i}"
								type="text"
								value={phase.label}
								oninput={(e) => updateLabel(i, (e.target as HTMLInputElement).value)}
							/>
						</td>
						<td>
							<label for="phase-start-{i}" class="sr-only">Phase {i + 1} start date</label>
							<input
								id="phase-start-{i}"
								type="date"
								value={phase.start_date}
								oninput={(e) => updateStart(i, (e.target as HTMLInputElement).value)}
							/>
						</td>
						<td>
							<button type="button" onclick={() => removePhase(i)} aria-label="Remove phase {i + 1}">×</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}

	<button type="button" onclick={addPhase}>Add phase</button>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}
	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{submitting ? 'Saving...' : 'Save Plan'}
	</button>
</form>

<style>
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
	}
	.generator { margin: var(--space-3, 1rem) 0; padding: var(--space-2, 0.5rem); }
	table { width: 100%; }
</style>
