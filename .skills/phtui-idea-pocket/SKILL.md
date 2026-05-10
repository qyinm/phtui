---
name: phtui-idea-pocket
description: Use when an agent needs low-token Product Hunt inspiration for service ideas, market scanning, feature references, or MVP positioning. Prefer phtui CLI JSON commands over MCP when possible.
version: 1.0.0
author: qyinm/phtui
license: MIT
metadata:
  tags: [product-hunt, ideas, cli, inspiration, market-research, agents]
---

# phtui Idea Pocket

## Overview

phtui is a Product Hunt-powered idea pocket for agents building new services.

Use the CLI first. MCP is available, but CLI JSON output is usually cheaper in tokens because the agent can request a narrow command, inspect only the fields it needs, and avoid loading a broad MCP tool context.

## Install

If `phtui` is already installed:

```bash
phtui help
```

Install latest from Go:

```bash
go install github.com/qyinm/phtui@latest
```

Or run from a checked-out repo:

```bash
go run . help
```

## Core Workflow

1. Start broad with `ideas`.
2. Pick 1-3 products that match the user's product direction.
3. Fetch details only for those selected products.
4. Summarize concrete reusable patterns: target user, core workflow, pricing clue, feature hook, and possible MVP angle.
5. Do not paste full raw JSON unless the user asks. Extract only the relevant signals.

## Commands

### Category inspiration

Use this when the user has a domain, category, or product area.

```bash
phtui ideas --category ai-agents --limit 5
```

From source checkout:

```bash
go run . ideas --category ai-agents --limit 5
```

Expected JSON shape:

```json
{
  "source": "category",
  "category_slug": "ai-agents",
  "total": 5,
  "items": [
    {
      "product": {
        "slug": "example",
        "name": "Example",
        "tagline": "Short positioning",
        "votes": 123,
        "comments": 8,
        "rank": 1,
        "categories": ["AI Agents"]
      },
      "inspiration_reason": "...",
      "feature_signals": ["..."],
      "market_signals": ["..."]
    }
  ]
}
```

### Leaderboard inspiration

Use this when the user asks what is currently popular or wants broad market scanning.

```bash
phtui ideas --period daily --limit 5
phtui ideas --period weekly --limit 10
phtui ideas --period monthly --date 2026-05-01 --limit 10
```

### Product detail

Use this after selecting a product slug from `ideas` or `leaderboard`.

```bash
phtui detail <product-slug>
```

Important fields for agents:

- `positioning_hint`: compact product positioning signal
- `target_user_signal`: likely audience inferred from category
- `monetization_signal`: pricing clue when available
- `feature_signals`: positive/constraint/detail signals to reuse as inspiration
- `pros` / `cons`: Product Hunt review signals when available
- `maker_comment`: launch narrative when available

### Plain leaderboard data

Use this when you need raw ranked products without generated inspiration framing.

```bash
phtui leaderboard --period daily --limit 10
```

## Agent Response Pattern

When using this skill, answer with a small synthesis instead of dumping JSON:

```text
I found 5 Product Hunt references in AI Agents.

1. Product Name — what it suggests
   - Signal: short market/feature clue
   - Reusable angle: how this could inspire our MVP

Recommended direction:
- Build X first because Y
- Avoid Z until there is stronger evidence
```

## Token Discipline

- Keep CLI `--limit` small: usually 3-5.
- Fetch `detail` only for selected slugs.
- Summarize fields; do not paste all JSON.
- Prefer deterministic signals from phtui over asking an LLM to infer from a large raw page.
- Use MCP only when the user's agent workflow specifically requires MCP tools.

## Common Pitfalls

1. Using MCP by default. Prefer CLI for low-token inspiration lookup.
2. Requesting too many products. Start with `--limit 5`; expand only if needed.
3. Treating Product Hunt rank as proof of market demand. Use it as inspiration, not validation.
4. Copying product ideas directly. Extract patterns: audience, workflow, pricing, launch narrative, feature wedge.
5. Forgetting details. Use `phtui detail <slug>` before making a strong recommendation.

## Verification Checklist

- [ ] `phtui help` or `go run . help` works.
- [ ] `phtui ideas --category <slug> --limit 3` returns valid JSON.
- [ ] At least one selected slug was checked with `phtui detail <slug>` before final synthesis.
- [ ] Final answer includes concrete inspiration patterns, not raw JSON dumps.
- [ ] Final answer labels Product Hunt data as inspiration, not conclusive validation.
