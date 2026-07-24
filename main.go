// Command tum is a sandy little app manager for the reMarkable.
//
// tum builds/installs apps onto a reMarkable over SSH as suffixed sibling
// appload entries (e.g. yaft-sandy) that never conflict with vellum-managed
// apps. See README.md for the full model.
package main

import "github.com/thecsw/tum/cmd"

func main() { cmd.Execute() }
