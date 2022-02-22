package static

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateStaticSample(t *testing.T) {
	p := NewProvider()
	require.NotNil(t, p)

	sample, _ := p.NextSample()
	require.Equal(t, []byte(stringSample), sample.Data)
}

func TestOnBindReturnsNil(t *testing.T) {
	p := NewProvider()
	err := p.OnBind()
	require.Nil(t, err)
}

func TestOnUnbindReturnsNil(t *testing.T) {
	p := NewProvider()
	err := p.OnUnbind()
	require.Nil(t, err)
}
