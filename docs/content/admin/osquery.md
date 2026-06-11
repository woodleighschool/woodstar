---
sidebar_position: 2
title: osquery
description: Reports, checks, and live queries against the fleet.
---

# osquery

osquery is how Woodstar asks the fleet questions. It's the replacement for Jamf's Extension Attributes and Smart Groups: write SQL, and either schedule it, turn it into a pass/fail check, or run it right now. It also drives dynamic labels and the software inventory.

This is Fleet's query model. A Woodstar report is a Fleet scheduled query; a Woodstar check is what Fleet calls a policy. The names are ours, the behaviour is theirs.

There are three things you author here.

## Reports

A report is a saved osquery query with a schedule and an optional label scope. Woodstar pushes it into the osquery config for the hosts in scope, and the results come back as snapshots you can look at over time.

Use a report when you want a recurring answer: installed apps, profiles, disk encryption status, whatever you'd have built an Extension Attribute for. Scope it with a label to ask only the machines it's relevant to.

## Checks

A check is a query turned into a pass or fail, the same idea as a Fleet policy. Each host that runs it lands on one side or the other, and Woodstar keeps the aggregate counts so you can see "47 passing, 3 failing" at a glance and drill into the three.

The query has to return enough for Woodstar to read a pass/fail result from it. The editor in the app is the place to get that shape right; the convention is best learned there rather than memorized from docs.

Use a check for things that should be true: a setting is enforced, an agent is running, a risky app isn't present.

## Live queries

A live query is a one-off. You pick a target, fire a query, and watch results stream in as hosts respond. Nothing is saved as a definition; it's for answering a question on the spot.

You can count the matching targets before you run, and stop a query that's still going. Results arrive through osquery's distributed write path and are tracked in memory while the query is live.

Endpoints for all three are in the [API reference](../api/overview). Retention, scheduling cadence, and how distributed reads and writes work on the wire are covered under [Agent Protocols](../agent-protocols/orbit-and-osquery).
