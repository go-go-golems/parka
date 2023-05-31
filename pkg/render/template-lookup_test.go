package render

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func ExampleLookupTemplateFromDirectory() {
	lookup := NewLookupTemplateFromDirectory("./templates")
	tmpl, err := lookup.Lookup("templates/tests/test.html")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Template:", tmpl.Name())
}

func ExampleLookupTemplateFromFS() {
	lookup := NewLookupTemplateFromFS(
		WithFS(os.DirFS("./templates")),
		WithBaseDir("./templates"),
		WithPatterns("*.html"),
	)
	tmpl, err := lookup.Lookup("tests/test.html")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Template:", tmpl.Name())
}

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

func TestLookupTemplateFromFS_SimpleDirectory(t *testing.T) {
	l := NewLookupTemplateFromFS(
		WithFS(os.DirFS("templates/tests")),
		WithPatterns("*.html"),
	)
	tmpl, err := l.Lookup("test.html")
	require.Nil(t, err)
	assert.Equal(t, "test.html", tmpl.Name())

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	require.Nil(t, err)
	assert.Equal(t, "foo foo", buf.String())

	tmpl, err = l.Lookup("test2.html")
	assert.NotNil(t, err)
}

func TestLookupTemplateFromFS_SimpleDirectoryWithBaseDir(t *testing.T) {
	l := NewLookupTemplateFromFS(
		WithFS(os.DirFS("templates")),
		WithBaseDir("tests"),
		WithPatterns("*.html"),
	)
	tmpl, err := l.Lookup("test.html")
	require.Nil(t, err)
	assert.Equal(t, "test.html", tmpl.Name())

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	require.Nil(t, err)
	assert.Equal(t, "foo foo", buf.String())

	tmpl, err = l.Lookup("test2.html")
	assert.NotNil(t, err)
}

func TestLookupTemplateFromFS_SimpleDirectoryWithReload(t *testing.T) {
	l := NewLookupTemplateFromFS(
		WithFS(os.DirFS("templates/tests")),
		WithPatterns("*.html"),
	)
	tmpl, err := l.Lookup("test.html")
	require.Nil(t, err)
	assert.Equal(t, "test.html", tmpl.Name())

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	require.Nil(t, err)
	assert.Equal(t, "foo foo", buf.String())

	// check that test_tmpl.html fails

	// change the template
	err = copyFile("templates/tests/test.html", "templates/tests/test_tmpl.html")
	require.NoError(t, err)

	// reload
	err = l.Reload()
	require.NoError(t, err)

	// check that test_tmpl.html succeeds
	tmpl, err = l.Lookup("test_tmpl.html")
	require.Nil(t, err)

	buf = new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	require.Nil(t, err)
	assert.Equal(t, "foo foo", buf.String())

	// check that test.html still succeeds
	tmpl, err = l.Lookup("test.html")
	require.Nil(t, err)

	buf = new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	require.Nil(t, err)
	assert.Equal(t, "foo foo", buf.String())

	// check that test2.html still fails
	tmpl, err = l.Lookup("test2.html")
	assert.NotNil(t, err)

	// delete test_tmpl.html
	err = os.Remove("templates/tests/test_tmpl.html")
	require.NoError(t, err)

	// reload
	err = l.Reload()
	require.NoError(t, err)

	// check that test_tmpl.html fails
	tmpl, err = l.Lookup("test_tmpl.html")
	assert.NotNil(t, err)

	// check that test.html still succeeds
	tmpl, err = l.Lookup("test.html")
	require.Nil(t, err)

	buf = new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	require.Nil(t, err)
	assert.Equal(t, "foo foo", buf.String())
}
