package contextsrv

import (
	"context"
)

type LLMClient interface {
	GenerateJSON(ctx context.Context, system, user string) (map[string]any, error)
}
