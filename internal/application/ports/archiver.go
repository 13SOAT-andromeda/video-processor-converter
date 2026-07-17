package ports

import "context"

type Archiver interface {
	// Zip compacta todos os arquivos de srcDir em destZipPath.
	Zip(ctx context.Context, srcDir, destZipPath string) error
}
