package resolver

import "testing"

func TestObservedVersions(t *testing.T) {
	v := ObservedVersions("node: v24.3.0\nSuccessfully set up CPython (3.13.15)\ngo version go1.26.3 linux/amd64")
	if v["node-version"] != "24.3.0" || v["python-version"] != "3.13.15" || v["go-version"] != "1.26.3" {
		t.Fatal(v)
	}
	if len(ObservedVersions("node: v20.1.0\nnode: v24.1.0")) != 0 {
		t.Fatal("ambiguous versions must not pin")
	}
}

func TestObservedUVVersionDoesNotFollowLatest(t *testing.T) {
	v := ObservedVersions("Successfully installed uv version 0.12.9")
	if v["uv-version"] != "0.12.9" {
		t.Fatal(v)
	}
	if v = ObservedVersions("Successfully installed uv version 0.12.9\nSuccessfully installed uv version 0.12.10"); v["uv-version"] != "" {
		t.Fatal("ambiguous versions", v)
	}
}
