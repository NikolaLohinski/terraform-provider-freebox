package providerdata

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type Setter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

type Getter interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}
