# Decisions

An ADR (architecture decision record) is a short note that records one decision about mhz:
what we chose, what we turned down, and why. Spikes produce drafts; the maintainer accepts
or rejects them. Number them in order (`0001-<slug>.md`), never rewrite an accepted one, and
supersede it with a new ADR instead.

## Template

```markdown
# NNNN. Title

Status: Proposed | Accepted | Rejected | Superseded by NNNN

## Context

What forces the decision. Facts only; mark each as measured, read in source, or assumed.

## Options

Each option considered, with its measurements and trade-offs.

## Decision

The option chosen, in one or two sentences.

## Consequences

What becomes easier, what becomes harder, and what is left open.
```
