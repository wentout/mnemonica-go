// Package foreign: a Sub whose parent handle is not a package-level
// Define/Sub in this package — mnemonica-gen only wires same-package
// handles and must say so.
package foreign

import (
	"mnemonica/mnemonica"
)

var collection = mnemonica.NewCollection()

type Widget struct {
	mnemonica.Node
	Name string
}

var WidgetT = mnemonica.Must(mnemonica.Define[Widget](collection, "Widget", func(a *Widget, name string) error {
	a.Name = name
	return nil
}))

type Gadget struct {
	mnemonica.Node
	*Widget
	SKU string
}

func foreign(parent *mnemonica.TypeDef[Widget, mnemonica.Root, string]) *mnemonica.TypeDef[Gadget, Widget, string] {
	gadgetT, _ := mnemonica.Sub[Gadget](parent, "Gadget", func(g *Gadget, sku string) error {
		g.SKU = sku
		return nil
	})
	return gadgetT
}

var ForeignT = foreign(WidgetT)

type Leaf struct {
	mnemonica.Node
	*Gadget
	Level int
}

var LeafT = mnemonica.Must(mnemonica.Sub[Leaf](ForeignT, "Leaf", func(l *Leaf, level int) error {
	l.Level = level
	return nil
}))
