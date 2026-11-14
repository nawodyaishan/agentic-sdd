// Spec 020 T12 / B5-W1-F — Q1 L1/U1/M1 sanitized recording templates.
// The templates are empty, consent-aware observations. An author run is always
// labelled `author_walkthrough` and can never establish unfamiliar-user proof.
// No real participant data, recording, transcript or contact detail belongs
// here; the emitted files are safe repository evidence only when sanitized.

import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';

export const PLAN_REFERENCE = 'plan §7 Q1 — Bounded usability and accessibility protocol';
export const AUTHOR_WALKTHROUGH = 'author_walkthrough';
export const UNFAMILIAR_PROOF_NOT_ESTABLISHED = 'unfamiliar-user proof not established';
export const FRESH_USER_UNVALIDATED = 'fresh-user onboarding unvalidated';

export const Q1_CARDS = Object.freeze({
	L1: {
		id: 'L1',
		person: 'Unfamiliar learner, not page author',
		timeboxMinutes: 20,
		task: 'From the index choose the core-only/client path; follow installation to isolated version/help/preview; explain changed files and stop before apply; identify scope/checks/unknowns in the direct-fix example.',
		observableResult: 'No real-home write, no accidental apply, and the participant can explain the effects.',
		failureDisposition: 'Real-home write, accidental apply or unclear effects is a blocking safety defect; no participant means not_run, not newcomer success.',
		defaultLabel: 'unfamiliar_user',
	},
	U1: {
		id: 'U1',
		person: 'Experienced agent user unfamiliar with these docs',
		timeboxMinutes: 15,
		task: 'Using navigation/search, find a rare flag/default, the shipped-versus-local profile distinction, the manual fallback and MCP recovery; record all four page IDs/headings.',
		observableResult: 'Four page IDs/headings are recorded without source or specification archaeology.',
		failureDisposition: 'Source/spec archaeology is a discoverability finding, not a user failure; no participant means not_run.',
		defaultLabel: 'experienced_user',
	},
	M1: {
		id: 'M1',
		person: 'Non-author maintainer if available',
		timeboxMinutes: 20,
		task: 'In an owned scratch tree, map CLI/config/skill changes to pages; add one page; break a link/anchor and interpret the file/reason diagnostics.',
		observableResult: 'One obvious edit owner, a local check command and actionable diagnostics.',
		failureDisposition: 'Ambiguous edit ownership or unactionable diagnostics is a docs-tooling defect; an author-only exercise is labelled author_walkthrough.',
		defaultLabel: 'maintainer',
	},
});

const LABELS = new Set([
	'unfamiliar_user',
	'experienced_user',
	'maintainer',
	AUTHOR_WALKTHROUGH,
]);
const STATUSES = new Set(['not_run', 'observed_satisfied', 'observed_issue', 'blocked']);
const REQUIRED_CONSENT_SCOPE = ['task observation', 'sanitized summary retention'];
const FORBIDDEN_KEYS = new Set([
	'name',
	'fullName',
	'firstName',
	'lastName',
	'email',
	'phone',
	'address',
	'employer',
	'handle',
	'ipAddress',
	'dateOfBirth',
	'exactAge',
	'recordingUrl',
	'transcript',
]);

function emptyResult() {
	return {
		completed: null,
		elapsedSeconds: null,
		pagesOrHeadings: [],
		wrongTurns: [],
		assistance: [],
		blockingDefects: [],
		notes: null,
	};
}

export function createRecording(cardId, options = {}) {
	const card = Q1_CARDS[cardId];
	if (!card) throw new Error(`unknown Q1 card: ${cardId}`);
	const label = options.label ?? AUTHOR_WALKTHROUGH;
	const template = {
		schemaVersion: 1,
		planReference: PLAN_REFERENCE,
		feature: '020-documentation-ecosystem',
		card: card.id,
		persona: card.person,
		timeboxMinutes: card.timeboxMinutes,
		label,
		observationStatus: 'not_run',
		consent: {
			obtained: false,
			recordedAt: null,
			scope: [],
			withdrawal: 'Participant may withdraw at any time; only the sanitized summary is retained.',
			retention: 'Sanitized summary only; no name, contact detail, recording or transcript.',
		},
		participant: {
			pseudonym: null,
			familiarity: null,
			isPageAuthor: label === AUTHOR_WALKTHROUGH,
		},
		session: {
			date: null,
			siteBase: null,
			sourceCommit: null,
			browser: null,
			assistiveTechnology: null,
			viewports: [
				{ widthCssPx: 320, zoomPercent: 100 },
				{ widthCssPx: 320, zoomPercent: 200 },
				{ widthCssPx: 1280, zoomPercent: 100 },
				{ widthCssPx: 1280, zoomPercent: 200 },
			],
		},
		result: emptyResult(),
		findings: [],
		limitations: [
			UNFAMILIAR_PROOF_NOT_ESTABLISHED,
			...(card.id === 'L1' || card.id === 'U1' ? [FRESH_USER_UNVALIDATED] : []),
		],
		unfamiliarUserProofEstablished: false,
		authorWalkthrough: label === AUTHOR_WALKTHROUGH,
		sanitized: true,
		containsParticipantIdentifiers: false,
	};
	return options.overrides ? { ...structuredClone(template), ...options.overrides } : structuredClone(template);
}

export function redactParticipantDetails(text) {
	return String(text)
		.replace(/[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/gi, '[redacted-email]')
		.replace(/(?:\+\d{1,3}[\s-]?)?(?:\(?\d{2,4}\)?[\s-]?){2,}\d{3,4}/g, '[redacted-phone]');
}

function walkKeys(value, visitor, pathParts = []) {
	if (Array.isArray(value)) {
		for (const [index, entry] of value.entries()) walkKeys(entry, visitor, [...pathParts, index]);
		return;
	}
	if (value && typeof value === 'object') {
		for (const [key, entry] of Object.entries(value)) {
			visitor(key, entry, [...pathParts, key]);
			walkKeys(entry, visitor, [...pathParts, key]);
		}
	}
}

export function validateRecording(recording) {
	if (!recording || typeof recording !== 'object') throw new Error('recording must be an object');
	if (recording.schemaVersion !== 1) throw new Error('unsupported schemaVersion');
	if (!Q1_CARDS[recording.card]) throw new Error(`unknown card: ${recording.card}`);
	if (!LABELS.has(recording.label)) throw new Error(`unknown label: ${recording.label}`);
	if (!STATUSES.has(recording.observationStatus)) throw new Error(`unknown status: ${recording.observationStatus}`);

	walkKeys(recording, (key, value, pathParts) => {
		if (FORBIDDEN_KEYS.has(key)) throw new Error(`participant-identifying key: ${pathParts.join('.')}`);
		if (typeof value !== 'string') return;
		if (/[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/i.test(value)) {
			throw new Error(`email-like value at ${pathParts.join('.')}`);
		}
		if (/(?:\+\d{1,3}[\s-]?)?(?:\(?\d{2,4}\)?[\s-]?){2,}\d{3,4}/.test(value)) {
			throw new Error(`phone-like value at ${pathParts.join('.')}`);
		}
	});

	if (recording.label === AUTHOR_WALKTHROUGH) {
		if (recording.unfamiliarUserProofEstablished) {
			throw new Error('an author_walkthrough cannot establish unfamiliar-user proof');
		}
		if (!recording.limitations.includes(UNFAMILIAR_PROOF_NOT_ESTABLISHED)) {
			throw new Error(`author_walkthrough must state "${UNFAMILIAR_PROOF_NOT_ESTABLISHED}"`);
		}
		if (recording.participant?.isPageAuthor !== true) {
			throw new Error('author_walkthrough must mark the participant as the page author');
		}
	}

	if (recording.unfamiliarUserProofEstablished) {
		if (!['L1', 'U1'].includes(recording.card)) {
			throw new Error('unfamiliar-user proof applies only to L1/U1');
		}
		if (!['observed_satisfied', 'observed_issue'].includes(recording.observationStatus)) {
			throw new Error('unfamiliar-user proof requires an observed status');
		}
		if (!recording.consent?.obtained) throw new Error('unfamiliar-user proof requires recorded consent');
	}

	if (recording.observationStatus === 'not_run') return true;

	const authorRun = recording.label === AUTHOR_WALKTHROUGH;
	if (!authorRun) {
		if (!recording.consent?.obtained) throw new Error('an observed run requires consent.obtained=true');
		for (const scope of REQUIRED_CONSENT_SCOPE) {
			if (!recording.consent.scope?.includes(scope)) throw new Error(`consent scope missing: ${scope}`);
		}
		if (!/^participant-[a-z0-9-]{1,24}$/.test(recording.participant?.pseudonym ?? '')) {
			throw new Error('observed runs require a pseudonymous participant id');
		}
		if (recording.label === 'unfamiliar_user' && recording.participant?.isPageAuthor !== false) {
			throw new Error('an unfamiliar participant cannot be the page author');
		}
	}
	if (typeof recording.result?.completed !== 'boolean') throw new Error('observed run requires result.completed');
	const elapsed = recording.result?.elapsedSeconds;
	if (!Number.isInteger(elapsed) || elapsed < 0 || elapsed > recording.timeboxMinutes * 60) {
		throw new Error('observed run requires elapsedSeconds inside the timebox');
	}
	return true;
}

function cell(value) {
	if (value === null || value === undefined) return 'not_run';
	if (Array.isArray(value)) return value.length ? value.map((entry) => String(entry).replace(/\|/g, '\\|')).join('; ') : 'not_run';
	return String(value).replace(/\|/g, '\\|');
}

export function renderRecordingMarkdown(recording) {
	validateRecording(recording);
	const card = Q1_CARDS[recording.card];
	const proof = recording.unfamiliarUserProofEstablished ? 'yes' : 'no';
	const reaffirm = recording.limitations.includes(UNFAMILIAR_PROOF_NOT_ESTABLISHED)
		? 'Unfamiliar-user proof not established: **yes**'
		: 'Unfamiliar-user proof not established: no';
	return `# Q1 ${card.id} observation record — ${card.person}

- Feature: \`${recording.feature}\`
- Card/timebox: ${card.id}, ${card.timeboxMinutes} minutes
- Label: \`${recording.label}\`
- Observation status: \`${recording.observationStatus}\`
- ${reaffirm}
- Unfamiliar-user proof established: ${proof}
- Sanitized: ${recording.sanitized ? 'yes' : 'no'} (no name, contact detail, recording or transcript)

## Consent

| Field | Value |
| --- | --- |
| Consent obtained | ${recording.consent.obtained ? 'yes' : 'no'} |
| Consent recorded at | ${cell(recording.consent.recordedAt)} |
| Scope | ${cell(recording.consent.scope)} |
| Withdrawal | ${cell(recording.consent.withdrawal)} |
| Retention | ${cell(recording.consent.retention)} |

## Participant

| Field | Value |
| --- | --- |
| Pseudonym | ${cell(recording.participant.pseudonym)} |
| Familiarity | ${cell(recording.participant.familiarity)} |
| Page author | ${recording.participant.isPageAuthor ? 'yes' : 'no'} |

## Session

| Field | Value |
| --- | --- |
| Date | ${cell(recording.session.date)} |
| Site base | ${cell(recording.session.siteBase)} |
| Source commit | ${cell(recording.session.sourceCommit)} |
| Browser | ${cell(recording.session.browser)} |
| Assistive technology | ${cell(recording.session.assistiveTechnology)} |
| Viewports | ${cell(recording.session.viewports.map((view) => `${view.widthCssPx}px @ ${view.zoomPercent}%`))} |

## Result

| Field | Value |
| --- | --- |
| Completed | ${cell(recording.result.completed)} |
| Elapsed seconds | ${cell(recording.result.elapsedSeconds)} |
| Pages/headings | ${cell(recording.result.pagesOrHeadings)} |
| Wrong turns | ${cell(recording.result.wrongTurns)} |
| Assistance | ${cell(recording.result.assistance)} |
| Blocking defects | ${cell(recording.result.blockingDefects)} |
| Notes | ${cell(recording.result.notes)} |

## Findings

${recording.findings.length ? recording.findings.map((finding) => `- ${finding}`).join('\n') : 'not_run'}

## Limitations

${recording.limitations.map((limitation) => `- ${limitation}`).join('\n')}
`;
}

export function emitTemplates(directory) {
	const written = [];
	for (const cardId of Object.keys(Q1_CARDS)) {
		const recording = createRecording(cardId);
		validateRecording(recording);
		const jsonPath = path.join(directory, `${cardId}.json`);
		const markdownPath = path.join(directory, `${cardId}.md`);
		writeFileSync(jsonPath, `${JSON.stringify(recording, null, 2)}\n`);
		writeFileSync(markdownPath, renderRecordingMarkdown(recording));
		written.push(jsonPath, markdownPath);
	}
	return written;
}

test('Q1 card definitions retain the approved person, timebox and failure disposition', () => {
	assert.deepEqual(Object.keys(Q1_CARDS), ['L1', 'U1', 'M1']);
	assert.equal(Q1_CARDS.L1.timeboxMinutes, 20);
	assert.equal(Q1_CARDS.U1.timeboxMinutes, 15);
	assert.equal(Q1_CARDS.M1.timeboxMinutes, 20);
	assert.match(Q1_CARDS.L1.person, /unfamiliar learner/i);
	assert.match(Q1_CARDS.U1.person, /experienced agent user/i);
	assert.match(Q1_CARDS.M1.person, /non-author maintainer/i);
	assert.match(Q1_CARDS.L1.failureDisposition, /blocking safety defect/);
	assert.match(Q1_CARDS.M1.failureDisposition, /author_walkthrough/);
});

test('default L1/U1/M1 records are sanitized not_run templates with consent fields', () => {
	for (const cardId of Object.keys(Q1_CARDS)) {
		const recording = createRecording(cardId);
		assert.doesNotThrow(() => validateRecording(recording));
		assert.equal(recording.observationStatus, 'not_run');
		assert.equal(recording.label, AUTHOR_WALKTHROUGH);
		assert.equal(recording.authorWalkthrough, true);
		assert.equal(recording.unfamiliarUserProofEstablished, false);
		assert.equal(recording.consent.obtained, false);
		assert.ok(recording.limitations.includes(UNFAMILIAR_PROOF_NOT_ESTABLISHED));
		if (cardId === 'L1' || cardId === 'U1') {
			assert.ok(recording.limitations.includes(FRESH_USER_UNVALIDATED));
		}
		assert.deepEqual(
			recording.session.viewports.map((view) => `${view.widthCssPx}@${view.zoomPercent}`),
			['320@100', '320@200', '1280@100', '1280@200'],
		);
	}
});

test('templates contain no participant identifiers and redact contact details', () => {
	for (const cardId of Object.keys(Q1_CARDS)) {
		const text = JSON.stringify(createRecording(cardId));
		assert.doesNotMatch(text, /[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/i);
		assert.doesNotMatch(text, /"(?:name|email|phone|address|transcript)"\s*:/i);
	}
	assert.equal(
		redactParticipantDetails('Contact learner@example.test or +44 20 7946 0958.'),
		'Contact [redacted-email] or [redacted-phone].',
	);
});

test('an author_walkthrough cannot establish unfamiliar-user proof', () => {
	const recording = createRecording('L1');
	recording.observationStatus = 'observed_satisfied';
	recording.consent.obtained = true;
	recording.consent.scope = [...REQUIRED_CONSENT_SCOPE];
	recording.consent.recordedAt = '2026-11-14';
	recording.participant.pseudonym = 'participant-author';
	recording.result.completed = true;
	recording.result.elapsedSeconds = 600;
	assert.doesNotThrow(() => validateRecording(recording));

	// An author walkthrough is a self-observation, not a consenting participant
	// session; it still cannot supply unfamiliar-user proof.
	const authorOnly = createRecording('L1');
	authorOnly.observationStatus = 'observed_issue';
	authorOnly.result.completed = false;
	authorOnly.result.elapsedSeconds = 420;
	authorOnly.findings = ['Heading wording was ambiguous.'];
	assert.doesNotThrow(() => validateRecording(authorOnly));

	recording.unfamiliarUserProofEstablished = true;
	assert.throws(
		() => validateRecording(recording),
		/author_walkthrough cannot establish unfamiliar-user proof/,
	);
});

test('an observed run requires consent and a pseudonymous participant', () => {
	const recording = createRecording('U1', { label: 'experienced_user' });
	recording.observationStatus = 'observed_satisfied';
	recording.result.completed = true;
	recording.result.elapsedSeconds = 300;
	assert.throws(() => validateRecording(recording), /requires consent/);

	recording.consent.obtained = true;
	recording.consent.scope = [...REQUIRED_CONSENT_SCOPE];
	recording.consent.recordedAt = '2026-11-14';
	assert.throws(() => validateRecording(recording), /pseudonymous participant/);

	recording.participant.pseudonym = 'participant-7';
	recording.participant.familiarity = 'uses another agent workflow daily';
	assert.doesNotThrow(() => validateRecording(recording));
});

test('emitted JSON and Markdown templates carry label, consent and limitation fields', (t) => {
	const directory = mkdtempSync(path.join(tmpdir(), 'spec020-q1-'));
	t.after(() => rmSync(directory, { recursive: true, force: true }));
	const written = emitTemplates(directory);
	assert.equal(written.length, 6);
	for (const cardId of Object.keys(Q1_CARDS)) {
		const json = JSON.parse(readFileSync(path.join(directory, `${cardId}.json`), 'utf8'));
		assert.equal(json.card, cardId);
		assert.equal(json.label, AUTHOR_WALKTHROUGH);
		assert.equal(json.unfamiliarUserProofEstablished, false);
		const markdown = readFileSync(path.join(directory, `${cardId}.md`), 'utf8');
		assert.match(markdown, /Unfamiliar-user proof not established: \*\*yes\*\*/);
		assert.match(markdown, /Label: `author_walkthrough`/);
		assert.match(markdown, /Consent obtained \| no/);
		assert.match(markdown, /Observation status: `not_run`/);
	}
});

test('real L1/U1/M1 sessions require consenting humans', { skip: 'requires consenting human participants; not_run' }, () => {
	assert.fail('not_run');
});
