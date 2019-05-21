package glue_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/gozix/glue/v2"
	glueMock "github.com/gozix/glue/v2/mock"
)

func TestBundles(t *testing.T) {
	type bundle struct {
		name      string
		dependsOn []string
	}

	var testCases = []struct {
		name        string
		bundles     []bundle
		expectError bool
	}{{
		name: "PositiveCase1",
		bundles: []bundle{
			{name: "a"},
			{name: "b"},
		},
	}, {
		name: "PositiveCase2",
		bundles: []bundle{
			{name: "a"},
			{name: "b"},
			{name: "c", dependsOn: []string{"a", "b"}},
		},
	}, {
		name: "NegativeCase1",
		bundles: []bundle{
			{name: "a"},
			{name: "a"},
		},
		expectError: true,
	}, {
		name: "NegativeCase2",
		bundles: []bundle{
			{name: "a"},
			{name: "b"},
			{name: "c", dependsOn: []string{"a", "c"}},
		},
		expectError: true,
	}, {
		name: "NegativeCase3",
		bundles: []bundle{
			{name: "a"},
			{name: "b"},
			{name: "c", dependsOn: []string{"a", "d"}},
		},
		expectError: true,
	}, {
		name: "NegativeCase4",
		bundles: []bundle{
			{name: "a"},
			{name: "b", dependsOn: []string{"c"}},
			{name: "c", dependsOn: []string{"a", "b"}},
		},
		expectError: true,
	}}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var bundles = make([]glue.Bundle, 0, len(testCase.bundles))
			for _, bundle := range testCase.bundles {
				if len(bundle.dependsOn) == 0 {
					var mocked = new(glueMock.Bundle)
					mocked.On("Name").Return(bundle.name)
					mocked.On("Build", mock.Anything).Return(nil)

					bundles = append(bundles, mocked)
					continue
				}

				var mocked = new(glueMock.BundleDependsOn)
				mocked.On("Name").Return(bundle.name)
				mocked.On("Build", mock.Anything).Return(nil)
				mocked.On("DependsOn").Return(bundle.dependsOn)

				bundles = append(bundles, mocked)
			}

			var _, err = glue.NewApp(
				glue.Bundles(bundles...),
			)

			if testCase.expectError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

	t.Run("NegativeCase5", func(t *testing.T) {
		var bundle = new(glueMock.Bundle)
		bundle.On("Name").Return("a")
		bundle.On("Build", mock.Anything).Return(errors.New("fake error"))

		var _, err = glue.NewApp(
			glue.Bundles(bundle),
		)
		assert.Error(t, err)
	})
}

func TestScopes(t *testing.T) {
	t.Run("PositiveCase1", func(t *testing.T) {
		var _, err = glue.NewApp(
			glue.Scopes("a", "b"),
		)
		assert.Nil(t, err)
	})

	t.Run("NegativeCase1", func(t *testing.T) {
		var _, err = glue.NewApp(
			glue.Scopes("a", "a"),
		)
		assert.Error(t, err)
	})
}

func TestExecute(t *testing.T) {
	var captureStdout = func(fn func() error) (_ []byte, err error) {
		var oStdout = *os.Stdout
		defer func() {
			*os.Stdout = oStdout
		}()

		var rStdout, wStdout *os.File
		if rStdout, wStdout, err = os.Pipe(); err != nil {
			return nil, err
		}

		*os.Stdout = *wStdout

		if err = fn(); err != nil {
			return nil, err
		}

		if err = wStdout.Close(); err != nil {
			return nil, err
		}

		var bStdout bytes.Buffer
		if _, err = io.Copy(&bStdout, rStdout); err != nil {
			return nil, err
		}

		if err = rStdout.Close(); err != nil {
			return nil, err
		}

		return bStdout.Bytes(), nil
	}

	var testCases = []struct {
		Name     string
		Args     []string
		Options  []glue.Option
		Contains string
	}{{
		Name:     "PositiveCase1",
		Args:     []string{"gozix-test-app"},
		Contains: "gozix-test-app",
	}, {
		Name: "PositiveCase2",
		Args: []string{"gozix-test-app", "version"},
		Options: []glue.Option{
			glue.Version("1.2.3"),
		},
		Contains: "1.2.3",
	}}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			var k, err = glue.NewApp(testCase.Options...)
			assert.Nil(t, err)

			var stdout []byte
			stdout, err = captureStdout(func() error {
				var oArgs = os.Args
				defer func() {
					os.Args = oArgs
				}()

				os.Args = testCase.Args

				return k.Execute()
			})

			assert.Nil(t, err)
			assert.True(t, strings.Contains(string(stdout), testCase.Contains))
		})
	}
}
