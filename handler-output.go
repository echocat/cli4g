package cli

import "sync"

// All framework help/error writes share this short-lived lock. Independent
// handlers may use the same non-concurrent writer. Rendering and callbacks do
// not run under this lock, so response -> help cannot recursively lock it.
// Custom/direct writer access still needs the caller's synchronization.
var handlerOutputMutex sync.Mutex
