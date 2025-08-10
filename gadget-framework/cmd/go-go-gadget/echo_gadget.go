package main

import (
    "context"
    "fmt"

    "inspector-gadget-os/gadget-framework/gadget"
)

type echoGadget struct{ gadget.NoopInstaller }

func newEchoGadget() *echoGadget { return &echoGadget{} }

func (g *echoGadget) Name() string        { return "echo" }
func (g *echoGadget) Description() string { return "Echoes provided arguments back to the console" }

func (g *echoGadget) Run(ctx context.Context, args []string) error {
    if len(args) == 0 {
        fmt.Println("(nothing to echo)")
        return nil
    }
    for _, a := range args {
        fmt.Println(a)
    }
    return nil
}


