module github.com/wentout/mnemonica-go/tools

go 1.27.0

require (
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	github.com/wentout/mnemonica-go v0.0.0-00010101000000-000000000000
	// The runtime require is not optional: cmd/mnemonica-gen's testdata
	// packages import github.com/wentout/mnemonica-go/mnemonica, and
	// `go mod tidy` cannot see testdata imports — do not drop this line.
	golang.org/x/tools v0.37.0
)

require (
	golang.org/x/mod v0.28.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/wentout/mnemonica-go => ../
