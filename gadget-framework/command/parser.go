package command

import (
    "context"
    "errors"
    "fmt"
    "strings"

    "inspector-gadget-os/gadget-framework/gadget"
)

// Parser provides a minimal CLI command parser for the gadget framework.
// Supported commands:
//   - list
//   - info <name>
//   - install <name>
//   - run <name> [args...]
//   - uninstall <name>
func Execute(ctx context.Context, mgr *gadget.Manager, args []string) error {
    if mgr == nil {
        return errors.New("manager is nil")
    }
    if len(args) == 0 {
        printHelp()
        return nil
    }
    cmd := args[0]
    switch cmd {
    case "list":
        infos := mgr.List()
        if len(infos) == 0 {
            fmt.Println("No gadgets registered.")
            return nil
        }
        for _, info := range infos {
            fmt.Printf("%-16s %s\n", info.Name, info.Description)
        }
        return nil
    case "info":
        if len(args) < 2 {
            return errors.New("usage: info <name>")
        }
        name := args[1]
        g, ok := mgr.Get(name)
        if !ok {
            return fmt.Errorf("unknown gadget: %s", name)
        }
        fmt.Printf("Name: %s\nDescription: %s\n", g.Name(), g.Description())
        return nil
    case "install":
        if len(args) < 2 {
            return errors.New("usage: install <name>")
        }
        return mgr.Install(ctx, args[1])
    case "run":
        if len(args) < 2 {
            return errors.New("usage: run <name> [args...]")
        }
        return mgr.Run(ctx, args[1], args[2:])
    case "uninstall":
        if len(args) < 2 {
            return errors.New("usage: uninstall <name>")
        }
        return mgr.Uninstall(ctx, args[1])
    case "help", "-h", "--help":
        printHelp()
        return nil
    default:
        fmt.Printf("Unknown command: %s\n\n", cmd)
        printHelp()
        return fmt.Errorf("unknown command: %s", cmd)
    }
}

func printHelp() {
    fmt.Println("Go Go Gadget CLI")
    fmt.Println()
    fmt.Println("Usage:")
    fmt.Println("  go-go-gadget <command> [args...]")
    fmt.Println()
    fmt.Println("Commands:")
    fmt.Println("  list                       List registered gadgets")
    fmt.Println("  info <name>               Show gadget details")
    fmt.Println("  install <name>            Install gadget")
    fmt.Println("  run <name> [args...]      Run gadget with optional args")
    fmt.Println("  uninstall <name>          Uninstall gadget")
    fmt.Println("  help                      Show this help")
    fmt.Println()
    fmt.Printf("Tip: %s\n", strings.TrimSpace("Use 'list' to discover available gadgets."))
}

