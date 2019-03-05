package embedder

/*
#include "flutter_embedder.h"

static char *c_str(uint8_t *str){
	return (char *)str;
}
*/
import "C"
import (
	"unsafe"

	"github.com/go-gl/glfw/v3.2/glfw"
)

// C proxies

//export proxy_platform_message_callback
func proxy_platform_message_callback(message *C.FlutterPlatformMessage, window unsafe.Pointer) {
	msg := &PlatformMessage{
		Channel: C.GoString(message.channel),
		Message: C.GoBytes(unsafe.Pointer(message.message), C.int(message.message_size)),

		ResponseHandle: PlatformMessageResponseHandle{
			cHandle: message.response_handle,
		},
	}
	index := *(*int)(glfw.GoWindow(window).GetUserPointer())
	flutterEngine := FlutterEngineByIndex(index)
	flutterEngine.FPlatfromMessage(msg)
}

//export proxy_make_current
func proxy_make_current(v unsafe.Pointer) C.bool {
	w := glfw.GoWindow(v)
	index := *(*int)(w.GetUserPointer())
	flutterEngine := FlutterEngineByIndex(index)
	return C.bool(flutterEngine.FMakeCurrent(v))
}

//export proxy_clear_current
func proxy_clear_current(v unsafe.Pointer) C.bool {
	w := glfw.GoWindow(v)
	index := *(*int)(w.GetUserPointer())
	flutterEngine := FlutterEngineByIndex(index)
	return C.bool(flutterEngine.FClearCurrent(v))
}

//export proxy_present
func proxy_present(v unsafe.Pointer) C.bool {
	w := glfw.GoWindow(v)
	index := *(*int)(w.GetUserPointer())
	flutterEngine := FlutterEngineByIndex(index)
	return C.bool(flutterEngine.FPresent(v))
}

//export proxy_fbo_callback
func proxy_fbo_callback(v unsafe.Pointer) C.uint32_t {
	w := glfw.GoWindow(v)
	index := *(*int)(w.GetUserPointer())
	flutterEngine := FlutterEngineByIndex(index)
	return C.uint32_t(flutterEngine.FFboCallback(v))
}

//export proxy_make_resource_current
func proxy_make_resource_current(v unsafe.Pointer) C.bool {
	w := glfw.GoWindow(v)
	index := *(*int)(w.GetUserPointer())
	flutterEngine := FlutterEngineByIndex(index)
	return C.bool(flutterEngine.FMakeResourceCurrent(v))
}

//export proxy_gl_proc_resolver
func proxy_gl_proc_resolver(v unsafe.Pointer, procname *C.char) unsafe.Pointer {
	return glfw.GetProcAddress(C.GoString(procname))
}
