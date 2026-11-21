//go:build android

package main

/*
#include <stdint.h>
#include <stdlib.h>
#include <jni.h>

// The screen-bright wake lock keeps the screen on while it is held. A wake lock
// needs no UI thread, unlike the window flag. Held once at a time.
static jobject tdWakeLock = NULL;

static void tdClearException(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
	}
}

void tdKeepScreenOn(uintptr_t envp, uintptr_t ctxp, int on) {
	JNIEnv *env = (JNIEnv *)envp;
	jobject ctx = (jobject)ctxp;
	if (on) {
		if (tdWakeLock != NULL) {
			return;
		}
		jclass ctxClass = (*env)->GetObjectClass(env, ctx);
		jmethodID getService = (*env)->GetMethodID(env, ctxClass, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;");
		jstring power = (*env)->NewStringUTF(env, "power");
		jobject pm = (*env)->CallObjectMethod(env, ctx, getService, power);
		tdClearException(env);
		if (pm == NULL) {
			return;
		}
		jclass pmClass = (*env)->GetObjectClass(env, pm);
		jmethodID newWakeLock = (*env)->GetMethodID(env, pmClass, "newWakeLock", "(ILjava/lang/String;)Landroid/os/PowerManager$WakeLock;");
		jstring tag = (*env)->NewStringUTF(env, "tomorrowdrop:screen");
		// SCREEN_BRIGHT_WAKE_LOCK: the screen stays on and bright.
		jobject lock = (*env)->CallObjectMethod(env, pm, newWakeLock, (jint)0x0000000a, tag);
		tdClearException(env);
		if (lock == NULL) {
			return;
		}
		jclass lockClass = (*env)->GetObjectClass(env, lock);
		jmethodID acquire = (*env)->GetMethodID(env, lockClass, "acquire", "()V");
		(*env)->CallVoidMethod(env, lock, acquire);
		tdClearException(env);
		tdWakeLock = (*env)->NewGlobalRef(env, lock);
		return;
	}
	if (tdWakeLock == NULL) {
		return;
	}
	jclass lockClass = (*env)->GetObjectClass(env, tdWakeLock);
	jmethodID release = (*env)->GetMethodID(env, lockClass, "release", "()V");
	(*env)->CallVoidMethod(env, tdWakeLock, release);
	tdClearException(env);
	(*env)->DeleteGlobalRef(env, tdWakeLock);
	tdWakeLock = NULL;
}
*/
import "C"

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

// setScreenAwake keeps the screen on (true) or lets it turn off (false).
func setScreenAwake(w fyne.Window, on bool) {
	native, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	flag := C.int(0)
	if on {
		flag = 1
	}
	native.RunNative(func(context any) {
		ac, ok := context.(*driver.AndroidWindowContext)
		if !ok {
			return
		}
		C.tdKeepScreenOn(C.uintptr_t(ac.Env), C.uintptr_t(ac.Ctx), flag)
	})
}
