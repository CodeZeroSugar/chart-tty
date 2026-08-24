package db

import (
	"errors"
	"path/filepath"
	"testing"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open() unexpected error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenCreatesFileWithSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "library.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() unexpected error: %v", err)
	}
	defer s.Close()

	if _, err := s.ListCharts(); err != nil {
		t.Errorf("ListCharts() on fresh library: %v", err)
	}
	if _, err := s.ListSetlists(); err != nil {
		t.Errorf("ListSetlists() on fresh library: %v", err)
	}
}

func TestAddAndGetChartRoundTrip(t *testing.T) {
	s := openTest(t)

	id, err := s.AddChart("Swing Low", "Chart TTY", "import", "{title: Swing Low}\n[G]body")
	if err != nil {
		t.Fatalf("AddChart() unexpected error: %v", err)
	}
	got, err := s.GetChart(id)
	if err != nil {
		t.Fatalf("GetChart(%d): %v", id, err)
	}
	if got.Title != "Swing Low" || got.Artist != "Chart TTY" || got.Source != "import" {
		t.Errorf("meta = %#v", got.ChartMeta)
	}
	if got.Content != "{title: Swing Low}\n[G]body" {
		t.Errorf("Content = %q", got.Content)
	}
}

func TestGetMissingChartErrors(t *testing.T) {
	s := openTest(t)
	if _, err := s.GetChart(999); err == nil {
		t.Fatal("GetChart(999) expected error")
	}
}

func TestListChartsNewestFirst(t *testing.T) {
	s := openTest(t)
	for _, title := range []string{"First", "Second", "Third"} {
		if _, err := s.AddChart(title, "", "import", "x"); err != nil {
			t.Fatal(err)
		}
	}
	charts, err := s.ListCharts()
	if err != nil {
		t.Fatalf("ListCharts(): %v", err)
	}
	wantOrder := []string{"Third", "Second", "First"}
	for i, want := range wantOrder {
		if charts[i].Title != want {
			t.Errorf("charts[%d].Title = %q, want %q", i, charts[i].Title, want)
		}
	}
}

func TestCreateSetlistDuplicateName(t *testing.T) {
	s := openTest(t)
	if _, err := s.CreateSetlist("Gig"); err != nil {
		t.Fatalf("CreateSetlist(): %v", err)
	}
	if _, err := s.CreateSetlist("Gig"); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate CreateSetlist() = %v, want ErrExists", err)
	}
}

func TestSetlistAppendAndOrderedFetch(t *testing.T) {
	s := openTest(t)
	slID, err := s.CreateSetlist("Sunday set")
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, title := range []string{"Opener", "Middle", "Closer"} {
		id, err := s.AddChart(title, "", "import", "content-"+title)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
		if err := s.AppendSetlistItem(slID, id); err != nil {
			t.Fatalf("AppendSetlistItem(): %v", err)
		}
	}

	charts, err := s.SetlistCharts(slID)
	if err != nil {
		t.Fatalf("SetlistCharts(): %v", err)
	}
	if len(charts) != 3 {
		t.Fatalf("got %d charts, want 3", len(charts))
	}
	for i, want := range []string{"Opener", "Middle", "Closer"} {
		if charts[i].Title != want {
			t.Errorf("charts[%d].Title = %q, want %q (performance order)", i, charts[i].Title, want)
		}
		if charts[i].Content != "content-"+want {
			t.Errorf("charts[%d].Content = %q", i, charts[i].Content)
		}
	}
	_ = ids
}

func TestSetlistListsInCreationOrder(t *testing.T) {
	s := openTest(t)
	for _, name := range []string{"A", "B"} {
		if _, err := s.CreateSetlist(name); err != nil {
			t.Fatal(err)
		}
	}
	lists, err := s.ListSetlists()
	if err != nil {
		t.Fatalf("ListSetlists(): %v", err)
	}
	if len(lists) != 2 || lists[0].Name != "A" || lists[1].Name != "B" {
		t.Errorf("lists = %#v", lists)
	}
}

func TestDeleteChartRemovesRow(t *testing.T) {
	s := openTest(t)
	id, _ := s.AddChart("Doomed", "", "import", "content")
	if err := s.DeleteChart(id); err != nil {
		t.Fatalf("DeleteChart(): %v", err)
	}
	if _, err := s.GetChart(id); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetChart after delete = %v, want ErrNotFound", err)
	}
}

func TestDeleteChartRemovesFromSetlist(t *testing.T) {
	s := openTest(t)
	id, _ := s.AddChart("Setlist Song", "", "import", "c")
	slID, _ := s.CreateSetlist("Gig")
	if err := s.AppendSetlistItem(slID, id); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteChart(id); err != nil {
		t.Fatal(err)
	}
	items, err := s.SetlistCharts(slID)
	if err != nil {
		t.Fatalf("SetlistCharts(): %v", err)
	}
	if len(items) != 0 {
		t.Errorf("setlist still has %d charts after deleting its only chart", len(items))
	}
}

func TestDeleteChartMissingReturnsNotFound(t *testing.T) {
	s := openTest(t)
	if err := s.DeleteChart(12345); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteChart(missing) = %v, want ErrNotFound", err)
	}
}

func TestDefaultPathUnderDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/data-here")
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath(): %v", err)
	}
	want := filepath.Join("/tmp/data-here", "chart-tty", "library.db")
	if path != want {
		t.Errorf("DefaultPath() = %q, want %q", path, want)
	}
}