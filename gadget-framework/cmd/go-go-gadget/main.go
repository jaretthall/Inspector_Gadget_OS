package main

import (
    "context"
    "os"

    "inspector-gadget-os/gadget-framework/command"
    "inspector-gadget-os/gadget-framework/gadget"
)

func main() {
    ctx := context.Background()

    mgr := gadget.NewManager()
    registerBuiltins(mgr)

    // Skip arg[0] which is the binary name
    _ = command.Execute(ctx, mgr, os.Args[1:])
}

// registerBuiltins registers built-in gadgets provided by the framework.
func registerBuiltins(mgr *gadget.Manager) {
    _ = mgr.Register(newEchoGadget())
    _ = mgr.Register(newSysInfoGadget())
}

