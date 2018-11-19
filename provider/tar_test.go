package provider

import "testing"

func TestSanitiseExtension(t *testing.T) {
	var inputs = []string{"gz", ".gz", "tar.gz", ".tar.gz"}
	var expected = "tar.gz"
	for _, ext := range inputs {
		out := sanitiseExtension(ext)
		if out != expected {
			t.Errorf("sanitiseExtension(\"%s\") -> %s, should be %s",
				ext, out, expected)
		}
	}
}
