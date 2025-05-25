package urlutil

import "net/url"

func Join(base *url.URL, paths ...string) *url.URL {
	u, err := url.Parse(base.String())
	if err != nil {
		panic("url copy failed: " + err.Error())
	}

	if u.Path == "" {
		u.Path = "/"
	}
	u.Path, err = url.JoinPath(u.Path, paths...)
	if err != nil {
		panic("url path join failed: " + err.Error())
	}

	return u
}
