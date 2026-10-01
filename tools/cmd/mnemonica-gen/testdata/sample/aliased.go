package sample

import m "mnemonica/mnemonica"

// External is declared through an ALIASED mnemonica import: detection goes
// through types (the PkgName), not the identifier, so this must still be
// generated.
type External struct {
	m.Node
	*Gadget
	SKU string
}

var ExternalT = m.Must(m.Sub[External](GadgetT, "External", func(e *External, sku string) error {
	e.SKU = sku
	return nil
}))
