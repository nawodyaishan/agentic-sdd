package skillsfiles

import "embed"

// AdapterFiles contains inactive resources only. Embedding never installs or
// discovers configuration and does not authorize a native/provider operation.
//
//go:embed integrations/opencode/opencode.jsonc integrations/opencode/policy-lockdown.jsonc integrations/opencode/model-usage.json harness/evidence/receipt.schema.json
var AdapterFiles embed.FS
