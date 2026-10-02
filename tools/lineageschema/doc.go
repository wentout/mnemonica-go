// Package lineageschema validates the runtime's lineage export against
// the shared JSON Schema (lineage.schema.json at the repo root). It lives
// in the tools module because schema validation needs the
// santhosh-tekuri/jsonschema dependency, and the runtime module stays
// stdlib-only. The tests import the runtime, produce graphs of every
// shape the format allows, and assert they validate.
package lineageschema
