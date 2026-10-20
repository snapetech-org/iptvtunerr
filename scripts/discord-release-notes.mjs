import path from 'node:path';
import { fileURLToPath } from 'node:url';

export function splitDiscordReleaseBody(body, maxLength = 3600) {
  if (typeof body !== 'string') {
    throw new TypeError('Release body must be a string.');
  }
  if (!Number.isInteger(maxLength) || maxLength < 1) {
    throw new RangeError('Maximum chunk length must be a positive integer.');
  }
  if (body.length === 0) {
    return [];
  }

  const chunks = [];
  let start = 0;
  while (start < body.length) {
    const limit = Math.min(start + maxLength, body.length);
    if (limit === body.length) {
      chunks.push(body.slice(start));
      break;
    }

    const newline = body.lastIndexOf('\n', limit - 1);
    let end = newline > start ? newline + 1 : limit;
    if (
      end < body.length &&
      end > start &&
      /[\uD800-\uDBFF]/u.test(body[end - 1]) &&
      /[\uDC00-\uDFFF]/u.test(body[end])
    ) {
      if (end === start + 1) {
        throw new RangeError(
          'Maximum chunk length cannot fit this Unicode character.'
        );
      }
      end -= 1;
    }

    chunks.push(body.slice(start, end));
    start = end;
  }

  return chunks;
}

export function formatDiscordReleaseBody(body) {
  if (typeof body !== 'string') {
    throw new TypeError('Release body must be a string.');
  }

  const lines = body.replace(/\r\n?/gu, '\n').split('\n');
  const start = lines.findIndex((line) => line.trim() === '## What Changed');
  if (start < 0) {
    throw new Error('Release body does not contain a What Changed section.');
  }

  let end = lines.findIndex(
    (line, index) =>
      index > start &&
      /^## (?:Included Commits|Release Assets|Install Notes)\s*$/u.test(line)
  );
  if (end < 0) {
    end = lines.length;
  }

  const summary = lines
    .slice(start + 1, end)
    .filter((line) => !/^\s*[_*]?Source:/iu.test(line))
    .join('\n')
    .trimStart()
    .replace(/^### User-facing changes\s*\n+/u, '')
    .replace(/^#### (.+?)\s*$/gmu, '### $1')
    .trim();

  if (!summary) {
    throw new Error('Release body has no user-facing changes to announce.');
  }

  return summary;
}

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) {
    chunks.push(chunk);
  }
  return Buffer.concat(chunks).toString('utf8');
}

const invokedPath = process.argv[1] ? path.resolve(process.argv[1]) : '';
if (invokedPath === fileURLToPath(import.meta.url)) {
  const args = process.argv.slice(2);
  if (args.length !== 1 || !['--format', '--split'].includes(args[0])) {
    process.stderr.write(
      'Usage: node scripts/discord-release-notes.mjs --format|--split\n'
    );
    process.exitCode = 2;
  } else {
    const body = await readStdin();
    if (args[0] === '--format') {
      process.stdout.write(`${formatDiscordReleaseBody(body)}\n`);
    } else {
      process.stdout.write(JSON.stringify(splitDiscordReleaseBody(body)) + '\n');
    }
  }
}
