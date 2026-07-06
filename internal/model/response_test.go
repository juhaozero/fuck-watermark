package model

import "testing"

func TestOKVideoData(t *testing.T) {
	data := NewVideoData(PlatformDouyin, MediaTypeVideo)
	data.Title = "test"
	resp := OK("解析成功", data)

	if resp.Code != 200 {
		t.Fatalf("code = %d", resp.Code)
	}
	vd, ok := resp.Data.(*VideoData)
	if !ok {
		t.Fatalf("data type = %T", resp.Data)
	}
	if vd.Platform != PlatformDouyin || vd.Type != MediaTypeVideo {
		t.Fatalf("unexpected data: %+v", vd)
	}
}

func TestBackupsFromURLs(t *testing.T) {
	backups := BackupsFromURLs("https://a", "", "https://b")
	if len(backups) != 2 {
		t.Fatalf("len = %d", len(backups))
	}
}
