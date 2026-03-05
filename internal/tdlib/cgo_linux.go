//go:build tde2e

package tdlib

// libtde2e.a exists in TDLib builds newer than commit 22d49d5 (go-tdlib v0.7.6).
// Build with -tags tde2e when your local TDLib install includes libtde2e.a.
// Docker builds use commit 22d49d5 which does not install libtde2e.a.

/*
#cgo linux LDFLAGS: -Wl,--whole-archive /usr/local/lib/libtde2e.a -Wl,--no-whole-archive
*/
import "C"
