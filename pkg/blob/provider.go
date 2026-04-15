package blob

import (
	"context"
)

// Provider define a interface soberana para armazenamento de objetos (blobs).
// Esta abstração permite que o Gojinn troque entre AWS S3, MinIO, Google Cloud Storage
// ou até mesmo o sistema de arquivos local sem alterar a lógica do runtime.
type Provider interface {
	// Put armazena os dados no provedor sob a chave (key) especificada.
	Put(ctx context.Context, key string, data []byte) error

	// Get recupera os dados associados à chave do provedor.
	Get(ctx context.Context, key string) ([]byte, error)

	// Close encerra quaisquer conexões ativas com o provedor (opcional, mas boa prática de SRE).
	Close() error
}
