package main

import (
	"flag"
	"net/url"
	"os"

	"github.com/iivvoo/clippan/clippan"
)

func NormalizeDSN(base string) (*url.URL, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	if u.Host == "" {
		u.Host = "localhost"
	}
	if u.Port() == "" {
		u.Host += ":5984"
	}
	// path (database) can be empty

	return u, nil
}

func main() {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Parse(os.Args[1:])
	dsn := flags.Arg(0)
	if dsn == "" {
		dsn = "http://admin:a-secret@localhost:5984"
	}
	dsnNormalized, err := NormalizeDSN(dsn)
	if err != nil {
		panic(err)
	}
	c := clippan.NewClippan(dsnNormalized.String())
	c.Run()
}
