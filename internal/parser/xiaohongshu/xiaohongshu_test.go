package xiaohongshu

import (
	"context"
	"encoding/json"

	"fmt"
	"fuck-watermark/internal/httputil"
	"fuck-watermark/internal/parser"
	"testing"
	"time"
)

func TestParser_Parse(t *testing.T) {
	client := httputil.New(30 * time.Second)
	p := New(client)
	resp := p.Parse(context.Background(), parser.Request{
		URL: "http://xhslink.com/o/92bFUni2jfG",
	})
	if resp.Code != 200 {
		t.Errorf("Parse failed: %v", resp.Msg)
	}
	jsonData, _ := json.Marshal(resp.Data)

	fmt.Println("parse xiaohongshu video response: ", string(jsonData))
	t.Logf("Parse response: %v", string(jsonData))

}
