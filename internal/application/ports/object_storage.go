package ports

import "context"

type ObjectStorage interface {
	// Exists faz HeadObject; retorna (false, nil) quando o objeto não existe (404),
	// (true, nil) quando existe, (false, err) em erro real.
	Exists(ctx context.Context, bucket, key string) (bool, error)
	// Download baixa o objeto para destPath no disco local (streaming).
	Download(ctx context.Context, bucket, key, destPath string) error
	// Upload envia srcPath para o bucket/key com o contentType informado (streaming).
	Upload(ctx context.Context, bucket, key, srcPath, contentType string) error
	// Delete remove o objeto.
	Delete(ctx context.Context, bucket, key string) error
}
