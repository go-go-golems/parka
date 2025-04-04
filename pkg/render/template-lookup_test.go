package render

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	_, err = l.Lookup("test2.txt")
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
	defer func() {
		_ = srcFile.Close()
	}()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = dstFile.Close()
	}()

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

	_, err = l.Lookup("test2.html")
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

	_, err = l.Lookup("test2.html")
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
	_, err = l.Lookup("test2.html")
	assert.NotNil(t, err)

	// delete test_tmpl.html
	err = os.Remove("templates/tests/test_tmpl.html")
	require.NoError(t, err)

	// reload
	err = l.Reload()
	require.NoError(t, err)

	// check that test_tmpl.html fails
	_, err = l.Lookup("test_tmpl.html")
	assert.NotNil(t, err)

	// check that test.html still succeeds
	tmpl, err = l.Lookup("test.html")
	require.Nil(t, err)

	buf = new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	require.Nil(t, err)
	assert.Equal(t, "foo foo", buf.String())
}

func TestLookupTemplateFromDirectory_PathTraversal(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "template-lookup-test")
	require.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temporary directory: %v", err)
		}
	}()

	// Create a test template file
	testTemplateContent := "<html>{{.Title}}</html>"
	err = os.WriteFile(filepath.Join(tempDir, "test.html"), []byte(testTemplateContent), 0644)
	require.NoError(t, err)

	// Create a "secret" file outside of the template directory that should not be accessible
	secretDir, err := os.MkdirTemp("", "secret-dir")
	require.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(secretDir); err != nil {
			t.Logf("Failed to remove secret directory: %v", err)
		}
	}()

	secretContent := "SECRET_DATA"
	secretFilePath := filepath.Join(secretDir, "secret.txt")
	err = os.WriteFile(secretFilePath, []byte(secretContent), 0644)
	require.NoError(t, err)

	// Create the lookup
	lookup := NewLookupTemplateFromDirectory(tempDir)

	// Test legitimate template access
	tmpl, err := lookup.Lookup("test.html")
	require.NoError(t, err)
	require.NotNil(t, tmpl)

	// Test path traversal attempts should fail
	traversalTests := []string{
		"../secret.txt",
		"../../secret.txt",
		"../secret-dir/secret.txt",
		"test.html/../../secret.txt",
		filepath.Join("..", filepath.Base(secretDir), "secret.txt"),
		filepath.Join("..", "..", "tmp", filepath.Base(secretDir), "secret.txt"),
	}

	for _, testPath := range traversalTests {
		t.Run(fmt.Sprintf("TestPathTraversal_%s", testPath), func(t *testing.T) {
			tmpl, err := lookup.Lookup(testPath)
			// Either it returns an error or nil template, but should never succeed
			if err == nil {
				require.Nil(t, tmpl, "Path traversal should not return a valid template")
			}
		})
	}
}
