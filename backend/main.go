package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func handler(fixture string) http.HandlerFunc {
	content, err := ioutil.ReadFile("./fixtures/" + fixture)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if s := r.URL.Query().Get("sleep"); s != "" {
			sleep, err := strconv.Atoi(s)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if sleep > 0 {
				time.Sleep(time.Duration(sleep) * time.Millisecond)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}
}

func main() {
	fixtures, err := ioutil.ReadDir("./fixtures")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	for _, f := range fixtures {
		mux.HandleFunc("/"+f.Name(), handler(f.Name()))
	}

	if err := http.ListenAndServe(":3000", mux); err != nil {
		fmt.Println(err)
	}
}
