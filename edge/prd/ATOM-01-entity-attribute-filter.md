# ATOM-01 — Expose `attributesContains` on entity and group queries

| | |
|---|---|
| **Repo** | `absmach/atom` (Rust) |
| **Priority** | P0 |
| **Depends on** | — |
| **Blocks** | MG-09, MG-15 (gateway → declared devices) |
| **Status** | Draft |

## Why this is P0

The device→gateway relation is stored as an attribute on the device
([spec §8 A10](../architecture.md#8-decision-record)):

```
Device meter-7 { gateways: ["gw-a", "gw-b"] }
```

"Which devices are declared on this gateway" is therefore exactly an
`attributesContains` query with JSONB array containment. Without it, the gateway
view has no declared-device list and the only fallback is fetching every device
in the domain and filtering client-side — which paginates incorrectly.

> This PRD was briefly demoted to P2 when an intermediate design stored the
> relation as a group. A10 reversed that; the justification is restored.

## Problem

Callers cannot filter entities or groups by attribute. `resources()` already
supports this; `entities()` and `groups()` do not, purely because the parameter
is hardcoded to `None` at the resolver.

Any product storing domain-specific state in `attributes` — which is what
`attributes` is for — currently has to fetch and filter client-side, which does
not paginate correctly and does not compose with authorization filtering.

## Why this is generic

This is a symmetry fix, not a feature. `Resource` and `Entity` both carry a
JSONB `attributes` column and both have list queries; only one can filter on it.
No caller-specific semantics are introduced — the filter is a containment check
over opaque JSON.

## Current state

| Layer | Status | Location |
|---|---|---|
| Query model | Field exists | `src/models/access.rs:98`, `src/models/resource.rs:51` |
| Repository | Implemented and bound | `src/authz/repo.rs:155,189,221` and `:5023,5139` |
| GraphQL — resources | **Exposed** | `src/graphql/resources.rs:51,75,108` |
| GraphQL — entities | Hardcoded `None` | `src/graphql/entities.rs:134` |
| GraphQL — groups | Hardcoded `None` | `src/graphql/groups.rs:263` |

## Scope

**In scope**

- Add `attributes_contains: Option<Value>` to the `entities()` query resolver
  (`src/graphql/entities.rs:74`), threading it through both the deleted-filter
  branch (`entities.rs:98`) and the live authorization-filtered branch
  (`entities.rs:123`).
- Same for `groups()` (`src/graphql/groups.rs:35`), threading through
  `authorized_group_list` (`groups.rs:214`).
- Mirror the parameter position and naming used by `resources()`
  (`src/graphql/resources.rs:44-56`) so the three queries stay consistent.

**Out of scope**

- Any change to the repository layer or SQL — the implementation already exists.
- New index work. Consider `GIN` on `entities.attributes` a follow-up, driven by
  measurement (see Risks).
- `authorizedObjectIds` — that is ATOM-02.

## Design

Follow `resources()` exactly. The parameter is `Option<Value>` (a JSON object),
passed to `ListEntities` / `AuthorizedObjectIdsQuery` unchanged. The repository
already filters out null values (`repo.rs:155`), so no resolver-side validation
is needed.

Ordering of parameters in the resolver signature should place
`attributes_contains` after `tenant_id`, matching `resources.rs:51`.

## Acceptance criteria

1. `entities(attributesContains: {provisioning_state: "pending"})` returns only
   entities whose attributes contain that pair.
1a. **Array containment works**: `entities(attributesContains: {gateways: ["gw-a"]})`
   returns devices whose `gateways` array *contains* `gw-a`, including those
   listing several. This is the gateway-view query and the reason for P0.
2. The filter composes with `kind`, `profileId`, `tenantId`, `parentGroupId`,
   `includeDescendants` and `status`.
3. The filter composes with authorization: a subject sees only entities they may
   read **and** that match the filter. Verify on the live branch
   (`entities.rs:123`), not just the platform-manage branch.
4. `total` in the returned `EntityList` reflects the filtered count, not the
   unfiltered one.
5. `groups(attributesContains: …)` behaves equivalently.
6. Omitting the argument produces byte-identical results to today.

## Test plan

- Unit: resolver passes the parameter through unchanged, including the `None`
  case.
- Integration: seed entities with differing attributes across two tenants; assert
  filtering, tenant isolation, pagination correctness (`total` and `offset`), and
  that an unauthorized subject sees nothing.
- Regression: existing `entities()` and `groups()` tests must pass untouched.

## Risks

- **Unindexed JSONB containment.** `attributes` has no `GIN` index. On large
  entity tables this filter will sequential-scan. Acceptable for the expected
  cardinality, but measure before relying on it in a hot path; add
  `CREATE INDEX ... USING GIN (attributes jsonb_path_ops)` if it shows up.
- **Filter composition with authorization** is the subtle part — the live branch
  routes through `authorized_object_ids`, so the filter must be applied inside
  that query rather than after it, or pagination silently breaks.
