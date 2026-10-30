package main

import "testing"

func TestRequiredSearchKindsAndProviderContract(t *testing.T) {
	s := testServer(t)
	for _, kind := range []string{"song", "album", "artist", "playlist", "all"} {
		w := call(t, s, "GET", "search?q=Artist&type="+kind, "")
		if w.Code != 200 {
			t.Fatalf("search %s: %d", kind, w.Code)
		}
	}
	if s.provider.Name() != "mock" {
		t.Fatalf("provider=%q", s.provider.Name())
	}
}
