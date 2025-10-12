package binder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_CookieBinder_Bind(t *testing.T) {
	t.Parallel()

	b := &cookieBinding{}
	require.Equal(t, "cookie", b.Name())

	type Post struct {
		Title string `form:"title"`
	}

	type User struct {
		Name  string   `form:"name"`
		Names []string `form:"names"`
		Posts []Post   `form:"posts"`
		Age   int      `form:"age"`
	}
	var user User

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "name", Value: "john"})
	req.AddCookie(&http.Cookie{Name: "names", Value: "john,doe"})
	req.AddCookie(&http.Cookie{Name: "age", Value: "42"})

	err := b.Bind(req, &user, true)

	require.NoError(t, err)
	require.Equal(t, "john", user.Name)
	require.Equal(t, 42, user.Age)
	require.Contains(t, user.Names, "john")
	require.Contains(t, user.Names, "doe")
}

func Benchmark_CookieBinder_Bind(b *testing.B) {
	b.ReportAllocs()

	binder := &cookieBinding{}

	type User struct {
		Name  string   `query:"name"`
		Posts []string `query:"posts"`
		Age   int      `query:"age"`
	}
	var user User

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "name", Value: "john"})
	req.AddCookie(&http.Cookie{Name: "age", Value: "42"})
	req.AddCookie(&http.Cookie{Name: "posts", Value: "post1,post2,post3"})

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

func Test_CookieBinder_Bind_ParseError(t *testing.T) {
	b := &cookieBinding{}
	type User struct {
		Age int `cookie:"age"`
	}
	var user User

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "age", Value: "invalid"})

	err := b.Bind(req, &user)
	require.Error(t, err)
}
