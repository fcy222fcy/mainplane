package harness

import (
	"context"
	"testing"
)

// fakeWorker is a worker over an in-memory map, enough for applyPatch.
type fakeWorker struct{ files map[string]string }

func (w fakeWorker) Run(context.Context, string, string, string, int) (Output, error) {
	return Output{}, nil
}
func (w fakeWorker) Job(context.Context, string, string, string, string, func(Output, error)) (int, string, error) {
	return 0, "", nil
}
func (w fakeWorker) Read(_ context.Context, path string) ([]byte, error) {
	b, ok := w.files[path]
	if !ok {
		return nil, errnoent
	}
	return []byte(b), nil
}
func (w fakeWorker) Write(_ context.Context, path string, data []byte) error {
	w.files[path] = string(data)
	return nil
}
func (w fakeWorker) Kill(context.Context, string) error { return nil }

// The blank line between *** End of File and *** End Patch is padding, not
// context: codex's parser ignores it (test_streaming_patch_parser_ignores_
// empty_lines_after_end_of_file), and models emit it. As context it demands a
// trailing empty line the file rarely has.
func TestParsePatchIgnoresEmptyLinesAfterEndOfFile(t *testing.T) {
	secs, err := parsePatch("*** Begin Patch\n*** Update File: f\n@@\n+quux\n*** End of File\n\n*** End Patch\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 1 || len(secs[0].hunks) != 1 {
		t.Fatalf("got %d sections, %d hunks", len(secs), len(secs[0].hunks))
	}
	h := secs[0].hunks[0]
	if !h.eof || len(h.old) != 0 || len(h.new) != 1 || h.new[0] != "quux" {
		t.Fatalf("eof %v, old %q, new %q", h.eof, h.old, h.new)
	}
}

// A blank line outside an EOF-marked hunk is still context.
func TestParsePatchKeepsEmptyContextWithoutEndOfFile(t *testing.T) {
	secs, err := parsePatch("*** Begin Patch\n*** Update File: f\n@@\n-a\n\n+b\n*** End Patch\n")
	if err != nil {
		t.Fatal(err)
	}
	h := secs[0].hunks[0]
	if h.eof || len(h.old) != 2 || h.old[1] != "" || len(h.new) != 2 || h.new[0] != "" || h.new[1] != "b" {
		t.Fatalf("eof %v, old %q, new %q", h.eof, h.old, h.new)
	}
}

func TestApplyPatchEndOfFileWithBlankLine(t *testing.T) {
	w := fakeWorker{files: map[string]string{"f": "hello\nworld\n"}}
	patch := "*** Begin Patch\n*** Update File: f\n@@\n+quux\n*** End of File\n\n*** End Patch\n"
	if err := applyPatch(context.Background(), w, patch); err != nil {
		t.Fatalf("applyPatch: %v", err)
	}
	if got := w.files["f"]; got != "hello\nworld\nquux\n" {
		t.Fatalf("got %q", got)
	}
}

var errnoent = errNotExist{}

type errNotExist struct{}

func (errNotExist) Error() string { return "file does not exist" }
