package ffprobe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Teste de caixa branca (package ffprobe, não ffprobe_test): parseProbeOutput
// não é exportada, e forjar um ffprobe real que devolva JSON inválido ou sem
// streams não é prático — testamos a função pura diretamente.

func TestParseProbeOutputInvalidJSON(t *testing.T) {
	_, err := parseProbeOutput([]byte("not json"))
	assert.Error(t, err)
}

func TestParseProbeOutputNoStreams(t *testing.T) {
	_, err := parseProbeOutput([]byte(`{"streams": []}`))
	assert.Error(t, err)
}
