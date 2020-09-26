package material

import (
	"unsafe"

	"git.wow.st/gmp/jni"
)

// #include <jni.h>
import "C"

var (
	// onConnect receives global IPNService references when
	// a VPN connection is requested.
	onConnect = make(chan jni.Object)
	// onDisconnect receives global IPNService references when
	// disconnecting.
	onDisconnect = make(chan jni.Object)
)

//export Java_com_lormann_leprechaun_ForegroundService_connect
func Java_com_lormann_leprechaun_ForegroundService_connect(env *C.JNIEnv, this C.jobject) {
	jenv := jni.EnvFor(uintptr(unsafe.Pointer(env)))
	onConnect <- jni.NewGlobalRef(jenv, jni.Object(this))
}

//export Java_com_lormann_leprechaun_ForegroundService_disconnect
func Java_com_lormann_leprechaun_ForegroundService_disconnect(env *C.JNIEnv, this C.jobject) {
	jenv := jni.EnvFor(uintptr(unsafe.Pointer(env)))
	onDisconnect <- jni.NewGlobalRef(jenv, jni.Object(this))
}
