// splunk-cli is a pipe-friendly CLI client for the Splunk REST API.
package main

import "github.com/nlink-jp/splunk-cli/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
