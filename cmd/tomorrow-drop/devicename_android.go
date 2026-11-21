//go:build android

package main

/*
#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>
#include <jni.h>

// tdStaticString reads a static String field of a system class through JNI.
// It returns NULL when the field cannot be read. Caller frees the result.
static char *tdStaticString(JavaVM *vm, const char *cls, const char *field) {
	JNIEnv *env = NULL;
	int attached = 0;
	if ((*vm)->GetEnv(vm, (void **)&env, JNI_VERSION_1_6) != JNI_OK) {
		if ((*vm)->AttachCurrentThread(vm, &env, NULL) != JNI_OK) {
			return NULL;
		}
		attached = 1;
	}
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
	if (attached) {
		(*vm)->DetachCurrentThread(vm);
	}
	return out;
}

// The NDK does not export JNI_GetCreatedJavaVMs to applications, but the
// runtime that hosts the application provides it, so it is looked up at run time.
typedef jint (*tdGetVMsFn)(JavaVM **, jsize, jsize *);

static JavaVM *tdJavaVM(void) {
	tdGetVMsFn get = (tdGetVMsFn)dlsym(RTLD_DEFAULT, "JNI_GetCreatedJavaVMs");
	if (get == NULL) {
		return NULL;
	}
	JavaVM *vm = NULL;
	jsize n = 0;
	if (get(&vm, 1, &n) != JNI_OK || n == 0) {
		return NULL;
	}
	return vm;
}

// tdManufacturer, tdModel and tdRelease read android.os.Build fields.
static char *tdManufacturer(void) { JavaVM *vm = tdJavaVM(); return vm ? tdStaticString(vm, "android/os/Build", "MANUFACTURER") : NULL; }
static char *tdModel(void)        { JavaVM *vm = tdJavaVM(); return vm ? tdStaticString(vm, "android/os/Build", "MODEL") : NULL; }
static char *tdRelease(void)      { JavaVM *vm = tdJavaVM(); return vm ? tdStaticString(vm, "android/os/Build$VERSION", "RELEASE") : NULL; }
*/
import "C"

import (
	"strings"
	"unsafe"
)

// devicePlatform returns the manufacturer and model, and the Android release.
// Manufacturer and model are joined into the name, so the name reads
// "samsung SM-S911B" and the system reads "Android 15".
func devicePlatform() (string, string) {
	manufacturer := readJava(C.tdManufacturer())
	model := readJava(C.tdModel())
	release := readJava(C.tdRelease())
	name := strings.TrimSpace(manufacturer + " " + model)
	system := ""
	if release != "" {
		system = "Android " + release
	}
	return name, system
}

func readJava(p *C.char) string {
	if p == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p)
}
