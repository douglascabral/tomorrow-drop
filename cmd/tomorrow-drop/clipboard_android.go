//go:build android

package main

/*
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <jni.h>

static void tdClipClear(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
	}
}


#define TD_MAX_FILE (64u << 20)

// tdReadStream reads a Java InputStream to the end into a malloc'd buffer. It
// fails when the stream throws or the file is larger than the limit.
int tdReadStream(JNIEnv *env, jobject in, unsigned char **out, size_t *n) {
	jclass inCls = (*env)->GetObjectClass(env, in);
	jmethodID read = (*env)->GetMethodID(env, inCls, "read", "([B)I");
	jmethodID close = (*env)->GetMethodID(env, inCls, "close", "()V");
	jbyteArray buf = (*env)->NewByteArray(env, 65536);
	size_t cap = 1u << 20, len = 0;
	unsigned char *data = malloc(cap);
	if (data == NULL || buf == NULL || read == NULL) {
		free(data);
		tdClipClear(env);
		return 0;
	}
	for (;;) {
		jint r = (*env)->CallIntMethod(env, in, read, buf);
		if ((*env)->ExceptionCheck(env)) {
			tdClipClear(env);
			free(data);
			return 0;
		}
		if (r < 0) {
			break;
		}
		if (r == 0) {
			continue;
		}
		if (len + (size_t)r > TD_MAX_FILE) {
			free(data);
			return 0;
		}
		if (len + (size_t)r > cap) {
			size_t next = cap * 2;
			unsigned char *grown = realloc(data, next);
			if (grown == NULL) {
				free(data);
				return 0;
			}
			data = grown;
			cap = next;
		}
		(*env)->GetByteArrayRegion(env, buf, 0, r, (jbyte *)(data + len));
		len += (size_t)r;
	}
	(*env)->CallVoidMethod(env, in, close);
	tdClipClear(env);
	*out = data;
	*n = len;
	return 1;
}

// tdReadUri reads a content Uri: its bytes through the content resolver, its
// display name from the provider (empty when it has none) and its MIME type.
int tdReadUri(JNIEnv *env, jobject ctx, jobject uri, unsigned char **out, size_t *n, char *name, size_t nameCap, char *mime, size_t mimeCap) {
	jobject cr = (*env)->CallObjectMethod(env, ctx, (*env)->GetMethodID(env, (*env)->GetObjectClass(env, ctx), "getContentResolver", "()Landroid/content/ContentResolver;"));
	tdClipClear(env);
	if (cr == NULL) {
		return 0;
	}
	jclass crCls = (*env)->GetObjectClass(env, cr);
	name[0] = 0;
	jobject cursor = (*env)->CallObjectMethod(env, cr, (*env)->GetMethodID(env, crCls, "query", "(Landroid/net/Uri;[Ljava/lang/String;Ljava/lang/String;[Ljava/lang/String;Ljava/lang/String;)Landroid/database/Cursor;"), uri, NULL, NULL, NULL, NULL);
	tdClipClear(env);
	if (cursor != NULL) {
		jclass curCls = (*env)->GetObjectClass(env, cursor);
		jboolean moved = (*env)->CallBooleanMethod(env, cursor, (*env)->GetMethodID(env, curCls, "moveToFirst", "()Z"));
		tdClipClear(env);
		if (moved) {
			jstring col = (*env)->NewStringUTF(env, "_display_name");
			jint index = (*env)->CallIntMethod(env, cursor, (*env)->GetMethodID(env, curCls, "getColumnIndex", "(Ljava/lang/String;)I"), col);
			tdClipClear(env);
			if (index >= 0) {
				jstring value = (jstring)(*env)->CallObjectMethod(env, cursor, (*env)->GetMethodID(env, curCls, "getString", "(I)Ljava/lang/String;"), index);
				tdClipClear(env);
				if (value != NULL) {
					const char *utf = (*env)->GetStringUTFChars(env, value, NULL);
					if (utf != NULL) {
						strncpy(name, utf, nameCap - 1);
						name[nameCap - 1] = 0;
						(*env)->ReleaseStringUTFChars(env, value, utf);
					}
				}
			}
		}
		(*env)->CallVoidMethod(env, cursor, (*env)->GetMethodID(env, curCls, "close", "()V"));
		tdClipClear(env);
	}
	mime[0] = 0;
	jstring type = (jstring)(*env)->CallObjectMethod(env, cr, (*env)->GetMethodID(env, crCls, "getType", "(Landroid/net/Uri;)Ljava/lang/String;"), uri);
	tdClipClear(env);
	if (type != NULL) {
		const char *t = (*env)->GetStringUTFChars(env, type, NULL);
		if (t != NULL) {
			strncpy(mime, t, mimeCap - 1);
			mime[mimeCap - 1] = 0;
			(*env)->ReleaseStringUTFChars(env, type, t);
		}
	}
	jobject in = (*env)->CallObjectMethod(env, cr, (*env)->GetMethodID(env, crCls, "openInputStream", "(Landroid/net/Uri;)Ljava/io/InputStream;"), uri);
	tdClipClear(env);
	if (in == NULL) {
		return 0;
	}
	return tdReadStream(env, in, out, n);
}

// tdClipItems returns how many items the primary clip has, or -1 when there is
// no clip.
int tdClipItems(uintptr_t envp, uintptr_t ctxp) {
	JNIEnv *env = (JNIEnv *)envp;
	jobject ctx = (jobject)ctxp;
	jclass ctxCls = (*env)->GetObjectClass(env, ctx);
	jstring name = (*env)->NewStringUTF(env, "clipboard");
	jobject cm = (*env)->CallObjectMethod(env, ctx, (*env)->GetMethodID(env, ctxCls, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;"), name);
	tdClipClear(env);
	if (cm == NULL) {
		return -1;
	}
	jclass cmCls = (*env)->GetObjectClass(env, cm);
	jobject clip = (*env)->CallObjectMethod(env, cm, (*env)->GetMethodID(env, cmCls, "getPrimaryClip", "()Landroid/content/ClipData;"));
	tdClipClear(env);
	if (clip == NULL) {
		return -1;
	}
	jclass clipCls = (*env)->GetObjectClass(env, clip);
	jint items = (*env)->CallIntMethod(env, clip, (*env)->GetMethodID(env, clipCls, "getItemCount", "()I"));
	tdClipClear(env);
	return (int)items;
}

// tdClipItem reads the item idx of the primary clip when it is a content Uri:
// its bytes, its display name and its MIME type. It returns 1 on success, and 0
// for an item that is not a file (text, for example).
int tdClipItem(uintptr_t envp, uintptr_t ctxp, int idx, unsigned char **out, size_t *n, char *name, size_t nameCap, char *mime, size_t mimeCap) {
	JNIEnv *env = (JNIEnv *)envp;
	jobject ctx = (jobject)ctxp;
	jclass ctxCls = (*env)->GetObjectClass(env, ctx);
	jstring cname = (*env)->NewStringUTF(env, "clipboard");
	jobject cm = (*env)->CallObjectMethod(env, ctx, (*env)->GetMethodID(env, ctxCls, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;"), cname);
	tdClipClear(env);
	if (cm == NULL) {
		return 0;
	}
	jclass cmCls = (*env)->GetObjectClass(env, cm);
	jobject clip = (*env)->CallObjectMethod(env, cm, (*env)->GetMethodID(env, cmCls, "getPrimaryClip", "()Landroid/content/ClipData;"));
	tdClipClear(env);
	if (clip == NULL) {
		return 0;
	}
	jclass clipCls = (*env)->GetObjectClass(env, clip);
	jobject item = (*env)->CallObjectMethod(env, clip, (*env)->GetMethodID(env, clipCls, "getItemAt", "(I)Landroid/content/ClipData$Item;"), (jint)idx);
	tdClipClear(env);
	if (item == NULL) {
		return 0;
	}
	jclass itemCls = (*env)->GetObjectClass(env, item);
	jobject uri = (*env)->CallObjectMethod(env, item, (*env)->GetMethodID(env, itemCls, "getUri", "()Landroid/net/Uri;"));
	tdClipClear(env);
	if (uri == NULL) {
		return 0;
	}
	return tdReadUri(env, ctx, uri, out, n, name, nameCap, mime, mimeCap);
}
*/
import "C"

import (
	"unsafe"

	"fyne.io/fyne/v2/driver"
)

// clipboardContents returns the files copied on the clipboard (for example in
// the Files app or a gallery), read through the content resolver. Text items are
// not files and are left out.
func clipboardContents() []sharedFile {
	var files []sharedFile
	_ = driver.RunNative(func(context any) error {
		ac, ok := context.(*driver.AndroidContext)
		if !ok {
			return nil
		}
		env, ctx := C.uintptr_t(ac.Env), C.uintptr_t(ac.Ctx)
		count := int(C.tdClipItems(env, ctx))
		for i := 0; i < count; i++ {
			var p *C.uchar
			var n C.size_t
			name := make([]C.char, 256)
			mime := make([]C.char, 128)
			if C.tdClipItem(env, ctx, C.int(i), &p, &n, &name[0], 256, &mime[0], 128) == 0 {
				continue
			}
			files = append(files, sharedFile{
				name: C.GoString(&name[0]),
				mime: C.GoString(&mime[0]),
				data: C.GoBytes(unsafe.Pointer(p), C.int(n)),
			})
			C.free(unsafe.Pointer(p))
		}
		return nil
	})
	return files
}

// clipboardPaths reports no paths on Android: files come as contents.
func clipboardPaths() []string {
	return nil
}

// clipboardImage reports no picture on Android: a picture is one of the files
// from clipboardContents.
func clipboardImage() ([]byte, string, bool) {
	return nil, "", false
}
