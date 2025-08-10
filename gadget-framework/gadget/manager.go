package gadget

import (
    "context"
    "errors"
    "fmt"
    "sort"
)

// Manager coordinates registration and lifecycle of gadgets.
type Manager struct {
    nameToGadget map[string]Gadget
}

func NewManager() *Manager {
    return &Manager{nameToGadget: make(map[string]Gadget)}
}

// Register adds a gadget to the manager. Returns error if name collides.
func (m *Manager) Register(g Gadget) error {
    if g == nil {
        return errors.New("gadget is nil")
    }
    name := g.Name()
    if name == "" {
        return errors.New("gadget name is empty")
    }
    if _, exists := m.nameToGadget[name]; exists {
        return fmt.Errorf("gadget %q already registered", name)
    }
    m.nameToGadget[name] = g
    return nil
}

// List returns gadget infos sorted by name.
func (m *Manager) List() []Info {
    infos := make([]Info, 0, len(m.nameToGadget))
    for _, g := range m.nameToGadget {
        infos = append(infos, Info{Name: g.Name(), Description: g.Description()})
    }
    sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
    return infos
}

func (m *Manager) Get(name string) (Gadget, bool) {
    g, ok := m.nameToGadget[name]
    return g, ok
}

func (m *Manager) Install(ctx context.Context, name string) error {
    g, ok := m.Get(name)
    if !ok {
        return fmt.Errorf("unknown gadget: %s", name)
    }
    return g.Install(ctx)
}

func (m *Manager) Run(ctx context.Context, name string, args []string) error {
    g, ok := m.Get(name)
    if !ok {
        return fmt.Errorf("unknown gadget: %s", name)
    }
    return g.Run(ctx, args)
}

func (m *Manager) Uninstall(ctx context.Context, name string) error {
    g, ok := m.Get(name)
    if !ok {
        return fmt.Errorf("unknown gadget: %s", name)
    }
    return g.Uninstall(ctx)
}


