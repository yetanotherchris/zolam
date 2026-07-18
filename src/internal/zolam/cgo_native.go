package zolam

// #cgo LDFLAGS: -L${SRCDIR}/../../native/tokenizers
import "C"

// This file exists solely to add native/tokenizers (populated by
// `go run ./tools/fetchnative`, see CLAUDE.md) to the linker search path,
// so github.com/daulet/tokenizers's `#cgo LDFLAGS: -ltokenizers` resolves
// without every developer having to export CGO_LDFLAGS by hand.
