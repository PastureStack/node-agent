package main

import "testing"

func TestTraditionalChineseOperatorMessage(t *testing.T) {
	if got := operatorMessage("zh-TW", "launch"); got != "正在啟動 PastureStack 節點代理程式" {
		t.Fatalf("unexpected zh-TW message: %q", got)
	}
}
