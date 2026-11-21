//go:build android

package main

/*
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <jni.h>

// tdBuildField reads a static String field of a system class (android.os.Build
// and its VERSION class) through JNI. It returns NULL when the field cannot be
// read. The caller frees the result. The environment comes from the application
// (driver.RunNative), which is attached to the Java runtime.
static char *tdBuildField(uintptr_t envp, const char *cls, const char *field) {
	JNIEnv *env = (JNIEnv *)envp;
	char *out = NULL;
	jclass c = (*env)->FindClass(env, cls);
	if (c != NULL) {
		jfieldID id = (*env)->GetStaticFieldID(env, c, field, "Ljava/lang/String;");
		if (id != NULL) {
			jstring s = (jstring)(*env)->GetStaticObjectField(env, c, id);
			if (s != NULL) {
				const char *utf = (*env)->GetStringUTFChars(env, s, NULL);
				if (utf != NULL) {
					out = strdup(utf);
					(*env)->ReleaseStringUTFChars(env, s, utf);
				}
				(*env)->DeleteLocalRef(env, s);
			}
		}
		(*env)->DeleteLocalRef(env, c);
	}
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
		free(out);
		out = NULL;
	}
	return out;
}
*/
import "C"

import (
	"strings"
	"unsafe"

	"fyne.io/fyne/v2/driver"
)

// devicePlatform returns the manufacturer and model, and the Android release.
// Manufacturer and model are joined into the name, so the name reads
// "motorola moto g75 5G" and the system reads "Android 15".
func devicePlatform() (string, string) {
	var manufacturer, model, release string
	_ = driver.RunNative(func(context any) error {
		ac, ok := context.(*driver.AndroidContext)
		if !ok {
			return nil
		}
		manufacturer = buildField(ac.Env, "android/os/Build", "MANUFACTURER")
		model = buildField(ac.Env, "android/os/Build", "MODEL")
		release = buildField(ac.Env, "android/os/Build$VERSION", "RELEASE")
		return nil
	})
	system := ""
	if release != "" {
		system = "Android " + release
	}
	return strings.TrimSpace(manufacturer + " " + model), system
}

func buildField(env uintptr, class, field string) string {
	cls, fld := C.CString(class), C.CString(field)
	defer C.free(unsafe.Pointer(cls))
	defer C.free(unsafe.Pointer(fld))
	p := C.tdBuildField(C.uintptr_t(env), cls, fld)
	if p == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p)
}
