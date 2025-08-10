package gadget

import "context"

// NoopInstaller provides default no-op lifecycle for simple gadgets.
type NoopInstaller struct{}

func (NoopInstaller) Install(ctx context.Context) error   { return nil }
func (NoopInstaller) Uninstall(ctx context.Context) error { return nil }


