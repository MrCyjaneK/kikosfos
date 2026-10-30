package main

import "testing"

func TestAppIcon(t *testing.T) {
	loadIcon()
	if iconW < 32 || iconH < 32 || len(iconPix) != iconW*iconH*4 {
		t.Fatalf("icon %dx%d len %d", iconW, iconH, len(iconPix))
	}
}
