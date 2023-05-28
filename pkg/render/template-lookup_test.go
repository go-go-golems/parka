package render

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLookupTemplateFromDirectory_SimpleDirectory(t *testing.T) {
	d := "templates/tests"
	l := NewLookupTemplateFromDirectory(d)
	tmpl, err := l.Lookup("test.txt")
	require.Nil(t, err)
	assert.Equal(t, "test.txt", tmpl.Name())

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	require.Nil(t, err)
	assert.Equal(t, "Foobar", buf.String())

	tmpl, err = l.Lookup("test2.txt")
	assert.NotNil(t, err)
}
