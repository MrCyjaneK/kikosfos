package main

/*
#cgo CFLAGS: -I${SRCDIR}
#define CARGADGETD_APP_IMPLEMENT_PLAY
#include "cg_app.h"
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"image"
	"unsafe"
)

var (
	gocarName = C.CString("Kiko")
	iconC     *C.uint8_t
)

func ensureIconC() {
	if iconC != nil {
		return
	}
	loadIcon()
	if len(iconPix) == 0 {
		return
	}
	n := C.size_t(len(iconPix))
	iconC = (*C.uint8_t)(C.malloc(n))
	if iconC == nil {
		return
	}
	C.memcpy(unsafe.Pointer(iconC), unsafe.Pointer(&iconPix[0]), n)
}

//export cargadgetd_app_gocar_icon
func cargadgetd_app_gocar_icon(w, h *C.int) *C.uint8_t {
	ensureIconC()
	if w != nil {
		*w = C.int(iconW)
	}
	if h != nil {
		*h = C.int(iconH)
	}
	return iconC
}

//export cargadgetd_app_gocar_init
func cargadgetd_app_gocar_init(_ *C.struct_cargadgetd_host) {
	gHUD.start()
}

//export cargadgetd_app_gocar_draw
func cargadgetd_app_gocar_draw(rgba *C.uint32_t, w, h, stride C.int) {
	if rgba == nil || w < 1 || h < 1 || stride < w*4 {
		return
	}
	pix := unsafe.Slice((*uint8)(unsafe.Pointer(rgba)), int(h)*int(stride))
	dst := &image.RGBA{
		Pix:    pix,
		Stride: int(stride),
		Rect:   image.Rect(0, 0, int(w), int(h)),
	}
	gHUD.ui.Lock()
	gHUD.ensure()
	gHUD.flush()
	gHUD.kit.Draw(dst)
	gHUD.ui.Unlock()
}

//export cargadgetd_app_gocar_touch
func cargadgetd_app_gocar_touch(x, y, action C.int) {
	gHUD.ui.Lock()
	gHUD.ensure()
	gHUD.kit.Touch(int(x), int(y), int(action))
	gHUD.flush()
	gHUD.ui.Unlock()
}

//export cargadgetd_app_gocar_pause
func cargadgetd_app_gocar_pause() { gHUD.kit.Pause() }

//export cargadgetd_app_gocar_resume
func cargadgetd_app_gocar_resume() { gHUD.kit.Resume() }

//export cargadgetd_app_gocar_name
func cargadgetd_app_gocar_name() *C.char { return gocarName }

func main() {}
