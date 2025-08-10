package gadget

import "context"

// Gadget defines the contract that every gadget must implement.
// Gadgets are small units of functionality that can be installed, run,
// and uninstalled by the framework.
type Gadget interface {
    // Name returns the unique name of the gadget.
    Name() string
    // Description provides a short human-readable description of the gadget.
    Description() string

    // Install performs any setup required for the gadget (idempotent if possible).
    Install(ctx context.Context) error
    // Run executes the gadget with the provided arguments.
    Run(ctx context.Context, args []string) error
    // Uninstall removes any resources created by the gadget (best effort).
    Uninstall(ctx context.Context) error
}

// Info contains metadata about a gadget.
type Info struct {
    Name        string `json:"name" yaml:"name"`
    Description string `json:"description" yaml:"description"`
}


