package tdlib

// libtde2e was added in TDLib 1.8.x; go-tdlib's LDFLAGS predate it.
// --whole-archive forces all symbols to be included regardless of link order.

/*
#cgo linux LDFLAGS: -Wl,--whole-archive /usr/local/lib/libtde2e.a -Wl,--no-whole-archive
*/
import "C"
