package compilerwasm

import (
	"context"

	"github.com/AricSu/tidb-clinic-client/compiler-rs/bindings/go/internal/assets"
)

func NewEmbedded(ctx context.Context) (*Client, error) {
	return New(ctx, assets.CompilerWasm)
}
