package cardigann

import "testing"

func TestDownloadBeforeRequestUsesDownloadURIQuery(t *testing.T) {
	d := &Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"download": map[string]any{"before": map[string]any{
		"path":   "thanks.php",
		"method": "post",
		"inputs": map[string]any{"infohash": "{{ .DownloadUri.Query.id }}", "thanks": 1},
	}}}}
	r := DownloadBeforeRequest(d, "https://idx.test/download.php?id=abc123&x=1")
	if r.Method != "post" || r.Path != "thanks.php" || r.Inputs["infohash"] != "abc123" || r.Inputs["thanks"] != "1" {
		t.Fatalf("got %#v", r)
	}
}

func TestDownloadBeforeRequestSupportsReReplaceOnPathAndQuery(t *testing.T) {
	d := &Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"download": map[string]any{"before": map[string]any{
		"path":   "takethanks.php",
		"method": "post",
		"inputs": map[string]any{"torrentid": `{{ re_replace .DownloadUri.PathAndQuery ".*download-torrent-(\d+).*" "$1"}}`},
	}}}}
	r := DownloadBeforeRequest(d, "https://idx.test/download-torrent-12345.html?foo=1")
	if r.Inputs["torrentid"] != "12345" {
		t.Fatalf("got %#v", r)
	}
}

func TestDownloadBeforeRequestSupportsAbsoluteURIPath(t *testing.T) {
	d := &Definition{BaseURL: "https://idx.test", Config: map[string]string{}, Raw: map[string]any{"download": map[string]any{"before": map[string]any{
		"path":   "{{ .DownloadUri.AbsoluteUri }}",
		"method": "post",
	}}}}
	r := DownloadBeforeRequest(d, "https://idx.test/download.php?id=abc123")
	if r.Path != "https://idx.test/download.php?id=abc123" {
		t.Fatalf("got %#v", r)
	}
}
