import assert from 'node:assert/strict';
import { execFileSync, spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const scriptsDirectory = path.dirname(fileURLToPath(import.meta.url));

const baselineSections = [
  { tag: 'v0.1.72', body: 'Published section for v0.1.72.' },
  { tag: 'v0.1.74', body: 'Published section for v0.1.74.' },
  { tag: 'v0.1.75', body: 'Published section for v0.1.75.' },
  { tag: 'v0.1.77', body: 'Published section for v0.1.77.' },
  { tag: 'v0.1.93', body: 'Published section for v0.1.93.' },
];

function renderChangelog(sections) {
  return [
    '# Changelog',
    '',
    ...sections.flatMap((section) => [
      `## [${section.tag}] - 2026-01-01`,
      '',
      section.body,
      '',
    ]),
  ].join('\n');
}

function runGit(repository, ...args) {
  execFileSync('git', args, { cwd: repository, stdio: 'ignore' });
}

function createFixture(t) {
  const repository = fs.mkdtempSync(
    path.join(os.tmpdir(), 'iptvtunerr-changelog-history-')
  );
  t.after(() => fs.rmSync(repository, { recursive: true, force: true }));

  const fixtureScripts = path.join(repository, 'scripts');
  const docs = path.join(repository, 'docs');
  fs.mkdirSync(fixtureScripts, { recursive: true });
  fs.mkdirSync(docs, { recursive: true });
  for (const file of [
    'check-changelog-tags.mjs',
    'changelog-tag-exceptions.json',
    'changelog-history-exceptions.json',
  ]) {
    fs.copyFileSync(
      path.join(scriptsDirectory, file),
      path.join(fixtureScripts, file)
    );
  }

  runGit(repository, 'init', '--quiet', '-b', 'main');
  runGit(repository, 'config', 'user.name', 'Release Test');
  runGit(repository, 'config', 'user.email', 'release-test@example.invalid');
  fs.writeFileSync(
    path.join(docs, 'CHANGELOG.md'),
    renderChangelog(baselineSections)
  );
  runGit(repository, 'add', '.');
  runGit(repository, 'commit', '--quiet', '-m', 'baseline changelog');
  for (const tag of [
    'v0.1.37',
    'v0.1.49',
    'v0.1.72',
    'v0.1.74',
    'v0.1.75',
    'v0.1.77',
    'v0.1.93',
  ]) {
    runGit(repository, 'tag', tag);
  }

  const failedCandidateSections = baselineSections.map((section) =>
    ['v0.1.74', 'v0.1.75', 'v0.1.77'].includes(section.tag)
      ? { ...section, body: `Accidentally edited ${section.tag}.` }
      : section
  );
  fs.writeFileSync(
    path.join(docs, 'CHANGELOG.md'),
    renderChangelog([
      ...failedCandidateSections,
      { tag: 'v0.1.94', body: 'Failed unpublished release candidate.' },
    ])
  );
  runGit(repository, 'add', '.');
  runGit(repository, 'commit', '--quiet', '-m', 'failed release candidate');
  runGit(repository, 'tag', 'v0.1.94');

  return { repository, docs, failedCandidateSections };
}

function runCheck(repository, base, head) {
  return spawnSync(
    process.execPath,
    [
      path.join(repository, 'scripts', 'check-changelog-tags.mjs'),
      base,
      head,
    ],
    { cwd: repository, encoding: 'utf8' }
  );
}

function addSuccessorCommit(fixture, sections) {
  fs.writeFileSync(
    path.join(fixture.docs, 'CHANGELOG.md'),
    renderChangelog([
      ...sections,
      { tag: 'v0.1.94', body: 'Failed unpublished release candidate.' },
      { tag: 'v0.1.95', body: 'Successor release section.' },
    ])
  );
  runGit(fixture.repository, 'add', '.');
  runGit(fixture.repository, 'commit', '--quiet', '-m', 'successor release');
  runGit(fixture.repository, 'tag', 'v0.1.95');
}

test('accepts only exact historical section restoration from the documented source tag', (t) => {
  const fixture = createFixture(t);
  addSuccessorCommit(fixture, baselineSections);

  const result = runCheck(fixture.repository, 'v0.1.94', 'v0.1.95');
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /Restored 3 published changelog section/);
});

test('does not apply the failed-tag exception before that tag is in the base history', (t) => {
  const fixture = createFixture(t);

  const result = runCheck(fixture.repository, 'v0.1.93', 'v0.1.94');
  assert.equal(result.status, 1);
  assert.match(
    result.stderr,
    /sections changed: v0\.1\.74, v0\.1\.75, v0\.1\.77/
  );
});

test('rejects edits to an allowlisted section that do not match its source tag', (t) => {
  const fixture = createFixture(t);
  const sections = fixture.failedCandidateSections.map((section) =>
    section.tag === 'v0.1.74'
      ? { ...section, body: 'A different historical edit.' }
      : section
  );
  addSuccessorCommit(fixture, sections);

  const result = runCheck(fixture.repository, 'v0.1.94', 'v0.1.95');
  assert.equal(result.status, 1);
  assert.match(result.stderr, /sections changed: v0\.1\.74/);
});

test('does not extend a recovery exception to an unrelated published section', (t) => {
  const fixture = createFixture(t);
  const sections = baselineSections.map((section) =>
    section.tag === 'v0.1.72'
      ? { ...section, body: 'Unrelated edit to a published release.' }
      : section
  );
  addSuccessorCommit(fixture, sections);

  const result = runCheck(fixture.repository, 'v0.1.94', 'v0.1.95');
  assert.equal(result.status, 1);
  assert.match(result.stderr, /sections changed: v0\.1\.72/);
});
