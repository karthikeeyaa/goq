package schema

import "embed"


// FS embeds all .sql files in the schema directory.
//go:embed *.sql
var FS embed.FS
