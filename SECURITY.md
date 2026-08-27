# Security policy

This library turns a qualified signature-validation report into one stable answer shape, so that
every service and user interface in a system renders the same verdict from the same place. That is
also its risk: it sits between a validator's report and everything a person reads, and a
normalisation that changes what the verdict says changes it for every consumer at once.

It validates nothing itself. The verdict belongs to the upstream validator; this code only reads
it.

Please report security problems privately. Do not open a public issue, pull request or discussion
for anything that could be exploited before a fix exists.

## How to report

Use **[private vulnerability reporting](https://github.com/gmb-lib/go-validation-answer/security/advisories/new)**
on this repository. The report stays visible only to you and the maintainers until an advisory is
published, and it gives us one place to discuss and co-ordinate a fix with you.

Please include, as far as you can establish it:

- what the problem is, and what an attacker or a mistaken reader gains from it;
- the smallest set of steps that reproduces it, and against which version or commit;
- the report bytes that trigger it, with anything sensitive removed;
- whether you have told anyone else, and whether a disclosure date already binds you.

## What happens next

- We acknowledge a report within **five working days**.
- We tell you whether we can reproduce it, and what we think its severity is, as soon as we know.
- We keep you updated while a fix is prepared, and we agree a disclosure date with you. Our default
  is to publish an advisory once a fix is available, and in any case within **90 days** of the
  report — earlier if the problem is already public or being exploited.
- We credit you in the advisory unless you would rather stay anonymous.

There is no bug-bounty programme. We are grateful anyway, and we say so publicly.

## What we consider most serious

- An answer that reports a signature as valid, qualified, or covering a document when the upstream
  report does not say so. Upgrading a verdict is the worst thing this library can do, and it does
  it everywhere at once.
- The quieter reverse: a good upstream verdict normalised into something a consumer renders as a
  failure, so a valid signature is rejected.
- Fallback resolution reading a field from the wrong place — a signer identity, timestamp or
  included-file entry taken from a layout that belongs to a different signature or a different
  document.
- An included-file list that under-reports what a signature actually covers, so a reader believes
  a document is signed when the signature never referenced it.
- Report bytes that cost unbounded time or memory to normalise.

Denial of service and findings that need an already-compromised host are in scope but lower
priority. This module depends on the standard library only, so there is no third-party dependency
surface to report.

## Scope

This policy covers the code in this repository. It does not cover the validation service whose
report is being normalised, or the signing API in front of it — report those to whoever operates
them.

## Supported versions

Security fixes land on the most recent release. Older tags are not patched; if you are pinned to
one, the fix is to move forward.
