package binder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_HeaderBinder_Bind(t *testing.T) {
	t.Parallel()

	b := &headerBinding{}
	require.Equal(t, "header", b.Name())

	type User struct {
		Name  string   `header:"Name"`
		Names []string `header:"Names"`
		Posts []string `header:"Posts"`
		Age   int      `header:"Age"`
	}
	var user User

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("name", "john")
	req.Header.Set("names", "john,doe")
	req.Header.Set("age", "42")
	req.Header.Set("posts", "post1,post2,post3")

	err := b.Bind(req, &user, true)

	require.NoError(t, err)
	require.Equal(t, "john", user.Name)
	require.Equal(t, 42, user.Age)
	require.Len(t, user.Posts, 3)
	require.Equal(t, "post1", user.Posts[0])
	require.Equal(t, "post2", user.Posts[1])
	require.Equal(t, "post3", user.Posts[2])
	require.Contains(t, user.Names, "john")
	require.Contains(t, user.Names, "doe")
}

func Benchmark_HeaderBinder_Bind(b *testing.B) {
	b.ReportAllocs()

	binder := &headerBinding{}

	type User struct {
		Name  string   `header:"Name"`
		Posts []string `header:"Posts"`
		Age   int      `header:"Age"`
	}
	var user User

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("name", "john")
	req.Header.Set("age", "42")
	req.Header.Set("posts", "post1,post2,post3")

	var err error
	for b.Loop() {
		err = binder.Bind(req, &user, true)
	}

	require.NoError(b, err)
	require.Equal(b, "john", user.Name)
	require.Equal(b, 42, user.Age)
	require.Len(b, user.Posts, 3)
	require.Contains(b, user.Posts, "post1")
	require.Contains(b, user.Posts, "post2")
	require.Contains(b, user.Posts, "post3")
}

func Test_HeaderBinder_Bind_ParseError(t *testing.T) {
	b := &headerBinding{}
	type User struct {
		Age int `header:"Age"`
	}
	var user User

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("age", "invalid")

	err := b.Bind(req, &user)
	require.Error(t, err)
}
