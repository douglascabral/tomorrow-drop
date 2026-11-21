//go:build android

package main

/*
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <jni.h>

// tdReadStream and tdReadUri are defined in clipboard_android.go.
int tdReadStream(JNIEnv *env, jobject in, unsigned char **out, size_t *n);
int tdReadUri(JNIEnv *env, jobject ctx, jobject uri, unsigned char **out, size_t *n, char *name, size_t nameCap, char *mime, size_t mimeCap);

static void tdShareClear1(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
	}
}

// tdShareStreams returns the stream Uris of the intent that opened the app: one
// for ACTION_SEND, a list for ACTION_SEND_MULTIPLE. It returns the activity's
// intent, or NULL.
static jobject tdShareIntent(JNIEnv *env, jobject ctx) {
	jclass ctxCls = (*env)->GetObjectClass(env, ctx);
	jmethodID getIntent = (*env)->GetMethodID(env, ctxCls, "getIntent", "()Landroid/content/Intent;");
	jobject intent = (*env)->CallObjectMethod(env, ctx, getIntent);
	tdShareClear1(env);
	return intent;
}

// tdShareInfo tells what the app was opened with to share. kind is 0 when
// nothing is shared, 1 for files (count of them) and 2 for text (into text,
// freed by the caller). It returns 1 when something is shared.
int tdShareInfo(uintptr_t envp, uintptr_t ctxp, int *kind, int *count, char **text) {
	JNIEnv *env = (JNIEnv *)envp;
	jobject ctx = (jobject)ctxp;
	*kind = 0;
	*count = 0;
	*text = NULL;
	jobject intent = tdShareIntent(env, ctx);
	if (intent == NULL) {
		return 0;
	}
	jclass inCls = (*env)->GetObjectClass(env, intent);
	jstring actionJ = (jstring)(*env)->CallObjectMethod(env, intent, (*env)->GetMethodID(env, inCls, "getAction", "()Ljava/lang/String;"));
	tdShareClear1(env);
	if (actionJ == NULL) {
		return 0;
	}
	const char *action = (*env)->GetStringUTFChars(env, actionJ, NULL);
	int send = action != NULL && strcmp(action, "android.intent.action.SEND") == 0;
	int multiple = action != NULL && strcmp(action, "android.intent.action.SEND_MULTIPLE") == 0;
	if (action != NULL) {
		(*env)->ReleaseStringUTFChars(env, actionJ, action);
	}
	if (!send && !multiple) {
		return 0;
	}
	jstring streamName = (*env)->NewStringUTF(env, "android.intent.extra.STREAM");
	if (multiple) {
		jobject list = (*env)->CallObjectMethod(env, intent, (*env)->GetMethodID(env, inCls, "getParcelableArrayListExtra", "(Ljava/lang/String;)Ljava/util/ArrayList;"), streamName);
		tdShareClear1(env);
		if (list == NULL) {
			return 0;
		}
		jclass listCls = (*env)->GetObjectClass(env, list);
		*count = (*env)->CallIntMethod(env, list, (*env)->GetMethodID(env, listCls, "size", "()I"));
		tdShareClear1(env);
		*kind = *count > 0 ? 1 : 0;
		return *kind;
	}
	jobject stream = (*env)->CallObjectMethod(env, intent, (*env)->GetMethodID(env, inCls, "getParcelableExtra", "(Ljava/lang/String;)Landroid/os/Parcelable;"), streamName);
	tdShareClear1(env);
	if (stream != NULL) {
		*kind = 1;
		*count = 1;
		return 1;
	}
	jstring textName = (*env)->NewStringUTF(env, "android.intent.extra.TEXT");
	jstring textJ = (jstring)(*env)->CallObjectMethod(env, intent, (*env)->GetMethodID(env, inCls, "getStringExtra", "(Ljava/lang/String;)Ljava/lang/String;"), textName);
	tdShareClear1(env);
	if (textJ == NULL) {
		return 0;
	}
	const char *utf = (*env)->GetStringUTFChars(env, textJ, NULL);
	if (utf == NULL) {
		return 0;
	}
	*text = strdup(utf);
	(*env)->ReleaseStringUTFChars(env, textJ, utf);
	*kind = 2;
	return *text != NULL;
}

// tdShareFile reads the file at index idx of the shared files: its bytes, its
// display name and its MIME type. It returns 1 on success.
int tdShareFile(uintptr_t envp, uintptr_t ctxp, int idx, unsigned char **out, size_t *n, char *name, size_t nameCap, char *mime, size_t mimeCap) {
	JNIEnv *env = (JNIEnv *)envp;
	jobject ctx = (jobject)ctxp;
	jobject intent = tdShareIntent(env, ctx);
	if (intent == NULL) {
		return 0;
	}
	jclass inCls = (*env)->GetObjectClass(env, intent);
	jstring streamName = (*env)->NewStringUTF(env, "android.intent.extra.STREAM");
	jobject uri = NULL;
	jobject list = (*env)->CallObjectMethod(env, intent, (*env)->GetMethodID(env, inCls, "getParcelableArrayListExtra", "(Ljava/lang/String;)Ljava/util/ArrayList;"), streamName);
	tdShareClear1(env);
	if (list != NULL) {
		jclass listCls = (*env)->GetObjectClass(env, list);
		uri = (*env)->CallObjectMethod(env, list, (*env)->GetMethodID(env, listCls, "get", "(I)Ljava/lang/Object;"), (jint)idx);
	} else {
		uri = (*env)->CallObjectMethod(env, intent, (*env)->GetMethodID(env, inCls, "getParcelableExtra", "(Ljava/lang/String;)Landroid/os/Parcelable;"), streamName);
	}
	tdShareClear1(env);
	if (uri == NULL) {
		return 0;
	}
	return tdReadUri(env, ctx, uri, out, n, name, nameCap, mime, mimeCap);
}

// tdShareClear empties the intent, so that the same files are not shared again
// when the app comes back to the foreground.
void tdShareClear(uintptr_t envp, uintptr_t ctxp) {
	JNIEnv *env = (JNIEnv *)envp;
	jobject ctx = (jobject)ctxp;
	jclass intentCls = (*env)->FindClass(env, "android/content/Intent");
	if (intentCls == NULL) {
		tdShareClear1(env);
		return;
	}
	jmethodID ctor = (*env)->GetMethodID(env, intentCls, "<init>", "()V");
	jobject empty = (*env)->NewObject(env, intentCls, ctor);
	jclass ctxCls = (*env)->GetObjectClass(env, ctx);
	(*env)->CallVoidMethod(env, ctx, (*env)->GetMethodID(env, ctxCls, "setIntent", "(Landroid/content/Intent;)V"), empty);
	tdShareClear1(env);
}
*/
import "C"

import (
	"unsafe"

	"fyne.io/fyne/v2/driver"
)

// readShare returns what other apps shared with Tomorrow Drop, once: the files
// (read through the content resolver) or the text. It clears the intent, so the
// same share is not offered again.
func readShare() (shareContent, bool) {
	var out shareContent
	found := false
	_ = driver.RunNative(func(context any) error {
		ac, ok := context.(*driver.AndroidContext)
		if !ok {
			return nil
		}
		env, ctx := C.uintptr_t(ac.Env), C.uintptr_t(ac.Ctx)
		var kind, count C.int
		var text *C.char
		if C.tdShareInfo(env, ctx, &kind, &count, &text) == 0 {
			return nil
		}
		switch kind {
		case 1:
			for i := 0; i < int(count); i++ {
				var p *C.uchar
				var n C.size_t
				name := make([]C.char, 256)
				mime := make([]C.char, 128)
				if C.tdShareFile(env, ctx, C.int(i), &p, &n, &name[0], 256, &mime[0], 128) == 0 {
					continue
				}
				out.files = append(out.files, sharedFile{
					name: C.GoString(&name[0]),
					mime: C.GoString(&mime[0]),
					data: C.GoBytes(unsafe.Pointer(p), C.int(n)),
				})
				C.free(unsafe.Pointer(p))
			}
		case 2:
			out.text = C.GoString(text)
			C.free(unsafe.Pointer(text))
		}
		C.tdShareClear(env, ctx)
		found = len(out.files) > 0 || out.text != ""
		return nil
	})
	return out, found
}
