package render

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
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

func TestLookupTemplateFromFS_DefaultValues(t *testing.T) {
	// the default values are . and *.html
	l := NewLookupTemplateFromFS()
	_, err := l.Lookup("test.txt")
	assert.Error(t, err)

	tmpl, err := l.Lookup("templates/tests/test.html")
	require.Nil(t, err)
	assert.Equal(t, "templates/tests/test.html", tmpl.Name())

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	require.Nil(t, err)
	assert.Equal(t, "foo foo", buf.String())
}

func TestLookupTemplateFromFS_Reload(t *testing.T) {
	// Setup
	l := NewLookupTemplateFromFS()
	testTemplatePath := "templates/tests/test.html"
	testTemplateCopyPath := "templates/tests/test_tmpl.html"

	_, err := l.Lookup(testTemplateCopyPath)
	assert.Error(t, err)

	// Copy test.html to test_tmpl.html
	err = copyFile(testTemplatePath, testTemplateCopyPath)
	require.NoError(t, err)

	// Call Reload and check that it resolves and has the correct content
	err = l.Reload()
	require.NoError(t, err)

	tmpl, err := l.Lookup(testTemplateCopyPath)
	require.NoError(t, err)
	assert.Equal(t, testTemplateCopyPath, tmpl.Name())

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	require.NoError(t, err)
	assert.Equal(t, "foo foo", buf.String())

	// Delete test_tmpl.html and call Reload again
	err = os.Remove(testTemplateCopyPath)
	require.NoError(t, err)

	err = l.Reload()
	require.NoError(t, err)

	_, err = l.Lookup(testTemplateCopyPath)
	assert.Error(t, err)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return dstFile.Sync()
}
