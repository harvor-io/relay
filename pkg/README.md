# pkg

Code here is kept decoupled from `internal/` on purpose: each package under
`pkg/` is self-contained (it manages its own schema, defines its own errors,
etc.) so that it could eventually be extracted into its own Go module and
reused outside of Relay, without dragging `internal/` along with it.

Until that happens, these packages are still consumed in-process by Relay
like any other package.
