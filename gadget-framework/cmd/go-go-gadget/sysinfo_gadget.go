package main

import (
    "context"
    "fmt"
    "runtime"
    "time"

    "inspector-gadget-os/gadget-framework/gadget"
)

type sysInfoGadget struct{ gadget.NoopInstaller }

func newSysInfoGadget() *sysInfoGadget { return &sysInfoGadget{} }

func (g *sysInfoGadget) Name() string        { return "sysinfo" }
func (g *sysInfoGadget) Description() string { return "Prints basic system information" }

func (g *sysInfoGadget) Run(ctx context.Context, args []string) error {
    fmt.Printf("Go: %s\n", runtime.Version())
    fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
    fmt.Printf("CPUs: %d\n", runtime.NumCPU())
    fmt.Printf("Now: %s\n", time.Now().Format(time.RFC3339))
    return nil
}


