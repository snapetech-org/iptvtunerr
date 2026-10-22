package store

import (
	"path/filepath"
	"testing"
)

func TestChannelAndStreamPaginationUsesBoundValues(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "pagination.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	for _, name := range []string{"Alpha", "Bravo", "Charlie"} {
		ch := &Channel{Name: name}
		if err := s.CreateChannel(ch); err != nil {
			t.Fatal(err)
		}
		stream := &Stream{ChannelID: &ch.ID, URL: "https://provider.example/" + name, Name: name}
		if err := s.CreateStream(stream); err != nil {
			t.Fatal(err)
		}
	}

	channels, total, err := s.ListChannels(ChannelListOpts{Page: 2, PerPage: 1})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(channels) != 1 || channels[0].Name != "Bravo" {
		t.Fatalf("channel page = %#v, total = %d", channels, total)
	}
	streams, total, err := s.ListStreams(StreamListOpts{Page: 2, PerPage: 1})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(streams) != 1 || streams[0].Name != "Bravo" {
		t.Fatalf("stream page = %#v, total = %d", streams, total)
	}

	maxInt := int(^uint(0) >> 1)
	channels, _, err = s.ListChannels(ChannelListOpts{Page: maxInt, PerPage: 2})
	if err != nil {
		t.Fatalf("overflow-sized channel page: %v", err)
	}
	if len(channels) != 0 {
		t.Fatalf("overflow-sized channel page returned rows: %#v", channels)
	}
	streams, _, err = s.ListStreams(StreamListOpts{Page: maxInt, PerPage: 2})
	if err != nil {
		t.Fatalf("overflow-sized stream page: %v", err)
	}
	if len(streams) != 0 {
		t.Fatalf("overflow-sized stream page returned rows: %#v", streams)
	}
}

func TestSafePageOffset(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, tc := range []struct {
		page, perPage, want int
	}{
		{page: -1, perPage: 20, want: 0},
		{page: 1, perPage: 20, want: 0},
		{page: 4, perPage: 20, want: 60},
		{page: maxInt, perPage: 2, want: maxInt},
	} {
		if got := safePageOffset(tc.page, tc.perPage); got != tc.want {
			t.Errorf("safePageOffset(%d, %d) = %d, want %d", tc.page, tc.perPage, got, tc.want)
		}
	}
}
