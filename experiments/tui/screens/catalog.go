// Package screens holds the experimental TUI screens.
package screens

import (
	"agentic-sdd/experiments/tui"
)

// Model is the catalog browser screen model.
type Model = tui.CatalogModel

// NewCatalog builds the browser for one read-only snapshot.
func NewCatalog(cfg tui.TUIConfig, catalog tui.Catalog) Model {
	return tui.NewCatalogModel(cfg, catalog)
}
