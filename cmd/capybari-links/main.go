// Command capybari-links runs this capability on its own.
package main

import (
	links "github.com/capybari-repo/capybari-analyzer-links"
	"github.com/capybari-repo/capybari-core/standalone"
)

var version = "dev"

func main() { standalone.Main(version, links.New()) }
