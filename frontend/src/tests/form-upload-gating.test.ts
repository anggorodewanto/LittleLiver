import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import SkinForm from '$lib/components/SkinForm.svelte';
import StoolForm from '$lib/components/StoolForm.svelte';
import AbdomenForm from '$lib/components/AbdomenForm.svelte';
import BruisingForm from '$lib/components/BruisingForm.svelte';
import NotesForm from '$lib/components/NotesForm.svelte';

// Regression: submit button must be disabled while a photo upload is in flight.
// Without this gate, users can submit before uploadPhoto() resolves, causing the
// log entry to be saved without the attached photo key(s).

const forms = [
	{ name: 'SkinForm', Component: SkinForm, defaultLabel: /log skin/i },
	{ name: 'StoolForm', Component: StoolForm, defaultLabel: /log stool/i },
	{ name: 'AbdomenForm', Component: AbdomenForm, defaultLabel: /log abdomen/i },
	{ name: 'BruisingForm', Component: BruisingForm, defaultLabel: /log bruising/i },
	{ name: 'NotesForm', Component: NotesForm, defaultLabel: /log note/i }
] as const;

describe('photo form submit gating', () => {
	for (const { name, Component, defaultLabel } of forms) {
		describe(name, () => {
			it('disables the submit button while a photo is uploading', () => {
				render(Component, {
					props: {
						onsubmit: vi.fn(),
						onphotoupload: vi.fn(),
						onphotoremove: vi.fn(),
						uploading: true
					}
				});

				const button = screen.getByRole('button', { name: /uploading/i }) as HTMLButtonElement;
				expect(button.disabled).toBe(true);
			});

			it('enables the submit button when not uploading or submitting', () => {
				render(Component, {
					props: {
						onsubmit: vi.fn(),
						onphotoupload: vi.fn(),
						onphotoremove: vi.fn(),
						uploading: false,
						submitting: false
					}
				});

				const button = screen.getByRole('button', { name: defaultLabel }) as HTMLButtonElement;
				expect(button.disabled).toBe(false);
			});
		});
	}
});
