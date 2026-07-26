package ports

import "context"

type FrameExtractor interface {
	// ExtractFrames extrai frames de inputPath para outDir e retorna a
	// quantidade de frames gerados.
	ExtractFrames(ctx context.Context, inputPath, outDir string) (int, error)
}
