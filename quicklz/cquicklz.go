package quicklz

/*
#cgo CFLAGS: -I .
#include "stdlib.h"
#include "quicklz.h"
size_t qlz_compress(const void *source, char *destination, size_t size, char *scratch_compress);
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/douban/gobeansdb/cmem"
	"github.com/douban/gobeansdb/utils"
)

const (
	CompressBufferSize   = 528400
	DecompressBufferSize = 16
)

func CCompress(src []byte) (dst cmem.CArray, ok bool) {
	ok = dst.Alloc(len(src) + 400)
	if !ok {
		return
	}
	buf := C.malloc(C.size_t(CompressBufferSize))
	if buf == nil {
		ok = false
		return
	}
	c_buf := (*C.char)(buf)
	defer C.free(unsafe.Pointer(c_buf))

	c_src := (unsafe.Pointer(&src[0]))
	c_dst := (*C.char)(unsafe.Pointer(&dst.Body[0]))
	c_size := C.qlz_compress(c_src, c_dst, C.size_t(len(src)), c_buf)
	size := int(c_size)
	dst.Body = dst.Body[:size]
	return
}

func CDecompress(src []byte, sizeD int) (dst cmem.CArray, err error) {
	if !dst.Alloc(sizeD) {
		err = fmt.Errorf("fail to alloc for decompress, size %d", sizeD)
		return
	}
	buf := make([]byte, DecompressBufferSize)
	c_src := (*C.char)(unsafe.Pointer(&src[0]))
	c_dst := (unsafe.Pointer(&dst.Body[0]))
	c_buf := (*C.char)(unsafe.Pointer(&buf[0]))
	size := int(C.qlz_decompress(c_src, c_dst, c_buf))
	if size != sizeD {
		err = fmt.Errorf("fail to alloc for decompress, size %d != %d", sizeD, size)
		return
	}
	dst.Body = dst.Body[:size]
	return
}

func DecompressSafe(src []byte) (dst []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			var ok bool
			err, ok = e.(error)
			if !ok {
				err = fmt.Errorf("decompress panic with non-error: %#v", e)
			}
		}
	}()
	sizeC := SizeCompressed(src)
	if len(src) != sizeC {
		return nil, fmt.Errorf("bad sizeCompressed, expect %d, got %d", sizeC, len(src))
	}
	sizeD := SizeDecompressed(src)
	dst = Decompress(src)
	if len(dst) != sizeD {
		return nil, fmt.Errorf("bad sizeDecompressed, expect %d, got %d", sizeD, len(dst))
	}
	return dst, nil
}

func CDecompressSafe(src []byte) (dst cmem.CArray, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("CDecompressSafe panic(%#v), stack: %s", e, utils.GetStack(1000))
		}
	}()
	sizeC := SizeCompressed(src)
	if len(src) != sizeC {
		err = fmt.Errorf("bad sizeCompressed, expect %d, got %d", sizeC, len(src))
		return
	}
	sizeD := SizeDecompressed(src)
	dst, err = CDecompress(src, sizeD)
	if err != nil {
		return
	}
	return
}
