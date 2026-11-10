---
paths:
  - "**/handlers/**/*.go"
  - "**/http/**/*.go"
  - "**/api/**/*.go"
  - "**/server/**/*.go"
  - "**/middleware/**/*.go"
  - "**/service/**/*.go"
  - "**/services/**/*.go"
  - "**/repositories/**"
  - "**/repository/**"
  - "**/store/**"
  - "**/static/docs/**"
  - "**/openapi*.y*ml"
---

# Go HTTP projects: pagination

General conventions for Go HTTP API projects. Where this project's own code or
CLAUDE.md differs, follow the project.

## Every list endpoint is paginated

- Every endpoint that returns a collection is paginated from its first
  version, at every stability level. Never "return everything for now".
- Pagination goes through `github.com/exalynt/turn`. Don't hand-roll limits,
  offsets, or cursors.
- Nothing matching is `200` with an empty array, `[]`. Never `null`, and never
  `404`.

## Pagination is never in the body

- The body is a bare JSON array of resources. No envelope, and no `data`,
  `items`, `next`, `cursor`, `has_more`, `page`, or `total` fields.
- Pagination information goes in standard headers:
  - `Link` ([RFC 8288](https://www.rfc-editor.org/rfc/rfc8288)), with
    `first`, `prev`, `next`, and `last` links, as far as each exists. Each
    link keeps the request's other query parameters, such as filters and
    `sort`. With no other pages, there is no `Link` header.
  - `X-Total-Count`, how many items match across every page, only on a
    listing whose clients need a total, such as a table that shows "42
    matching". It costs a `COUNT` query, so don't add it to every listing.
    Like the pages, the count can drift as items change between requests.
- Add both headers to CORS's exposed headers
  (`Access-Control-Expose-Headers`), or browser clients can't read them.

## Cursors by default

- Use cursor pagination, with the `after`, `before`, and `size` query
  parameters. Use numbered pages (`page`, `size`) only for a small or bounded
  listing where a UI needs to jump to page _N_.
- Page sizes are turn's defaults: 25 per page, 100 at most. A size out of
  range, a malformed cursor, or `after` and `before` together is a `400`,
  never clamped or silently restarted from the first page.
- Cursors are opaque. Clients take them from the `Link` header and never build
  them.
- A `sort` query parameter names a field, with a leading `-` for descending.
  An unknown field is a `400`. Every order ends with a unique key, such as the
  ID, so the order is total and no item is skipped or repeated between pages.

## Through the layers

**Handlers** parse the selection, call the service, and set the headers:

- A shared `cursorSelector(w, r)` parses the selection with
  `turnhttp.CursorQuery`, writing the `400` itself when it's invalid.
- After the service returns the page, a shared `setCursorLinks(w, r, page)`
  sets `Link` with `turnhttp.FormatLinks(turnhttp.CursorLinks(page,
  cursorQuery.URL(r.URL))...)`, then the handler sets `X-Total-Count` if the
  listing has one, and renders `page.Items` as resources.
- `ErrInvalidPage` maps to `400`, with a message that says what was wrong
  (the size's range, both directions, or an invalid cursor), without echoing
  the cursor back.

**Services** take a `cursor.Selector` alongside the listing's filter, pass it
into the repository's query, and return the `cursor.Page[models.X]`, plus the
total when the listing counts. They translate turn's selector errors into
their own: when `paginator.IsSelectorError(err)`, return
`fmt.Errorf("%w: %w", ErrInvalidPage, err)`. Any other error is a `500`.

**Repositories** own the query and its pagination:

- `List(ctx, query XQuery) (cursor.Page[models.X], error)`, where `XQuery`
  holds the filters, `Sort`, and `Page cursor.Selector`. When the listing
  counts, `Count(ctx, query)` takes the same query and ignores `Page` and
  `Sort`.
- Build the paginator once per listing and order, as a package-level value,
  with `store.NewCursor` and `.Reverse()` for the descending order. Render
  the keyset `WHERE`, `ORDER BY`, and `LIMIT` with `turnsql.Render`. The base
  query, filters, and tenant scoping stay the repository's own SQL.
- Bind each cursor to its listing: pass the listing's name and the query's
  filters as scope values, so a cursor replayed against another listing or
  other filters is rejected, not a wrong page. Leave the caller's
  authorization bounds out of the scope (`json:"-"`). They aren't a filter
  the client chose.
- Keyset columns are non-null.

## Documenting

- In the OpenAPI spec, list endpoints use shared `after`, `before`, and `size`
  parameters and a shared `Link` response header, plus `X-Total-Count` where
  the listing has one.
- The endpoint's description says to follow the `Link` header's `next` link
  for the rest.

## Changing a listing

Changing pagination follows the REST API rule's "Changing endpoints". At Beta
and GA, adding pagination to a listing that returned everything is breaking,
since existing clients would only see the first page.
