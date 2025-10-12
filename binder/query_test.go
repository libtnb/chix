package binder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_QueryBinder_Bind(t *testing.T) {
	t.Parallel()

	b := &queryBinding{}
	require.Equal(t, "query", b.Name())

	type Post struct {
		Title string `query:"title"`
	}

	type User struct {
		Name  string   `query:"name"`
		Names []string `query:"names"`
		Posts []Post   `query:"posts"`
		Age   int      `query:"age"`
	}
	var user User

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.URL.RawQuery = "name=john&names=john,doe&age=42&posts[0][title]=post1&posts[1][title]=post2&posts[2][title]=post3"

	err := b.Bind(req, &user, true)

	require.NoError(t, err)
	require.Equal(t, "john", user.Name)
	require.Equal(t, 42, user.Age)
	require.Len(t, user.Posts, 3)
	require.Equal(t, "post1", user.Posts[0].Title)
	require.Equal(t, "post2", user.Posts[1].Title)
	require.Equal(t, "post3", user.Posts[2].Title)
	require.Contains(t, user.Names, "john")
	require.Contains(t, user.Names, "doe")
}

func Benchmark_QueryBinder_Bind(b *testing.B) {
	b.ReportAllocs()

	binder := &queryBinding{}

	type User struct {
		Name  string   `query:"name"`
		Posts []string `query:"posts"`
		Age   int      `query:"age"`
	}
	var user User

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.URL.RawQuery = "name=john&age=42&posts=post1,post2,post3"

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
