# Org Clipper Native Messaging Protocol (schemaVersion 1)

Single source of truth for messages between the extension service worker and
the native host `com.karohani.org_clipper`. Both sides validate every field.

Transport: Chrome Native Messaging over a long-lived port (`runtime.connectNative`).
Framing: 4-byte native-endian (little-endian on macOS) length + UTF-8 JSON.

## Limits

| Item | Limit |
|------|-------|
| Incoming message (extension -> host) | 8 MiB, checked from the length prefix BEFORE reading the body |
| Outgoing message (host -> extension) | 64 KiB (Chrome's hard limit is 1 MB) |
| `title` | 1..1000 characters, single line after normalisation |
| `sourceUrl` | <= 4096 chars, `http:` or `https:` only |
| `blocks` total text | <= 4 MiB; total node count <= 50,000; nesting depth <= 12 |
| `assets` | <= 30 entries, each `url` <= 4096 chars |
| Per image | <= 10 MiB received bytes, <= 50 megapixels, each side <= 20,000 px |
| Per clip, all images | <= 50 MiB received bytes |

## Request: `saveClip`

```json
{
  "schemaVersion": 1,
  "type": "saveClip",
  "requestId": "8c3f0f8e-1f8a-4c55-9a59-0e5a3f8f9d21",
  "category": "clippings",
  "title": "Original page title",
  "sourceUrl": "https://example.com/post?id=1",
  "clippedAt": "2026-11-08T10:00:00+09:00",
  "downloadAssets": true,
  "blocks": [],
  "assets": [ { "id": "a1", "url": "https://cdn.example.com/x.png" } ]
}
```

- `requestId`: lowercase UUID v4 (`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`). Retries reuse the same id.
- `category`: one of the IDs below; anything else is rejected with `unknown_category`.
- `clippedAt`: RFC 3339 timestamp.
- `assets[].id`: `^a[0-9]{1,3}$`, unique. Every image node references an id from this list.
- When `downloadAssets` is false the host downloads nothing; image nodes render as a plain-text placeholder.

| Category ID | Folder (relative to the save root) |
|-------------|------------------------------------|
| `articles` | `00. Inbox/01. Articles` |
| `papers` | `00. Inbox/02. Papers` |
| `transcripts` | `00. Inbox/03. Transcripts` |
| `clippings` | `00. Inbox/04. Clippings` |
| `ai-research` | `00. Inbox/05. AI Research` |

### Block nodes (`t` field)

| `t` | Fields |
|-----|--------|
| `heading` | `level` 1..6, `inl: Inline[]` |
| `para` | `inl: Inline[]` |
| `list` | `ordered: bool`, `items: Block[][]` |
| `quote` | `blocks: Block[]` |
| `table` | `header: bool` (first row is a header), `rows: Inline[][][]` (row -> cell -> inlines) |
| `code` | `text: string` (rendered as an escaped `example` block, never `src`) |
| `image` | `asset: string` |
| `hr` | — |

### Inline nodes (`t` field)

| `t` | Fields |
|-----|--------|
| `text` | `v: string` |
| `bold` / `italic` / `strike` | `c: Inline[]` |
| `code` | `v: string` |
| `link` | `href: string`, `c: Inline[]` (only `http:`/`https:` hrefs become links; others render as text) |
| `image` | `asset: string` |
| `br` | — |

Unknown `t` values, unknown fields, wrong types, or exceeded limits reject the whole request with `invalid_request`.

## Request: `status`

```json
{ "schemaVersion": 1, "type": "status", "requestId": "<uuid>" }
```

Returns the stored `result` for a previously processed request, or a `result` with `status: "unknown"`.

## Request: `ping`

```json
{ "schemaVersion": 1, "type": "ping" }
```

Returns `{ "type": "pong", "schemaVersion": 1, "configured": true, "version": "x.y.z" }`. Never returns the root path.

## Responses

1. After a `saveClip` passes validation: `{ "type": "ack", "requestId": "<uuid>" }` — the request was delivered, nothing is saved yet.
2. After the files are in their final location, or on failure:

```json
{
  "type": "result",
  "requestId": "<uuid>",
  "status": "saved",
  "path": "00. Inbox/04. Clippings/2026-11-08-원문-제목.org",
  "assetsSaved": 3,
  "failedAssets": [ { "id": "a2", "reason": "blocked_address" } ],
  "duplicate": false,
  "error": { "code": "disk_full", "message": "No space left on device" }
}
```

| `status` | Meaning |
|----------|---------|
| `saved` | Document and every requested image stored |
| `partial` | Document stored; some images failed (`failedAssets`) |
| `failed` | Nothing stored (`error` set) |
| `in_progress` | Another host process is handling the same `requestId` |
| `unknown` | `status` request for an id the host has no record of |

- `duplicate: true` when the result is replayed from the request ledger instead of being produced now.
- `path` is relative to the save root; the root itself is never sent.

Error codes: `invalid_request`, `unknown_category`, `not_configured`, `root_unavailable`, `permission_denied`, `disk_full`, `too_large`, `internal`.

Asset failure reasons: `not_https`, `unsupported_scheme` (data:/blob:/file:), `blocked_address`, `dns_failed`, `too_many_redirects`, `http_status`, `timeout`, `too_large`, `total_limit`, `count_limit`, `unsupported_type`, `bad_signature`, `too_many_pixels`, `decode_failed`, `network`, `skipped` (downloadAssets=false).

## Output layout

```
<root>/00. Inbox/<category folder>/
  <name>.org                    name = <YYYY-MM-DD>-<slug>, or <name>-2, -3, ... if taken
  assets/<name>/<first 32 hex of sha256>.<png|jpg|webp>   only when an image was saved
<root>/.org-clipper/            host-private state (dot-prefixed)
  tmp/<requestId>-<random>/     staging; assets moved first, document linked last
  requests/<requestId>.json     result ledger (idempotent retries)
  requests/<requestId>.lock     in-flight marker
```
