#ifndef CARGADGETD_CG_APP_H
#define CARGADGETD_CG_APP_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/*
 * Plugin ABI: each .so exports C symbols named
 *
 *   cargadgetd_app_<id>_<method>
 *
 * <id> is a C identifier (the app id). The host dlsyms these after
 * discovering <id> from the .so (dynamic symbol *_draw, else the <id> in
 * the filename). When both are present they must be the same id.
 *
 * Library name must be `libcargadgetd_app_<id>.so`, installed in
 * `/usr/lib/cargadgetd/plugins/`. Other names (music.so, reversegearhead.so)
 * are not apps. The helloworld package's id is `hello`.
 *
 *   void        cargadgetd_app_<id>_init(const struct cargadgetd_host *host);
 *   void        cargadgetd_app_<id>_draw(uint32_t *rgba, int w, int h, int stride_bytes);
 *   void        cargadgetd_app_<id>_touch(int x, int y, int action); // 0 up, 1 down, 2 move
 *   void        cargadgetd_app_<id>_pause(void);
 *   void        cargadgetd_app_<id>_resume(void);
 *   void        cargadgetd_app_<id>_stop(void); // optional; host Close / plugin unload
 *   const char *cargadgetd_app_<id>_name(void);
 *   const uint8_t *cargadgetd_app_<id>_icon(int *w, int *h); // packed RGBA8; C.malloc (host copies)
 *   void        cargadgetd_app_<id>_huinput_<event>(int pressed); // optional; 1 down, 0 up
 *
 * HU keys are broadcast to every loaded app (opened or not). <event> names:
 *   mic menu home back call endcall key0..key9 star pound
 *   dpad_up dpad_down dpad_left dpad_right dpad_click plus search
 *   playpause stop next prev play pause
 *   dpad_up_left dpad_down_left dpad_up_right dpad_down_right
 *   rotary media navigation tel
 * Unknown scan codes are not called; they go to apps.unhandled_log.
 *
 * Host (passed to init):
 *   play      — optional PCM mix into the shell
 *   frame_ms  — paint + video interval; 0 → 33 (same knob as video.frame_ms)
 *
 * Required: draw. All others optional. name/icon may return NULL.
 */

struct cargadgetd_host {
	void (*play)(const int16_t *pcm, int nframes, int channels, int sample_rate);
	int frame_ms; /* paint + video interval; 0 → 33 */
};

#ifdef CARGADGETD_APP_IMPLEMENT_PLAY
static void cargadgetd_host_play(const struct cargadgetd_host *h,
				 const int16_t *pcm, int nframes, int channels, int sample_rate)
{
	if (h && h->play)
		h->play(pcm, nframes, channels, sample_rate);
}

static int cargadgetd_host_frame_ms(const struct cargadgetd_host *h)
{
	int ms;

	if (!h || h->frame_ms <= 0)
		return 33;
	ms = h->frame_ms;
	if (ms < 16)
		return 16;
	return ms;
}
#endif

#ifdef __cplusplus
}
#endif

#endif
