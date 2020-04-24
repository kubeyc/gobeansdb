package store

import (
	"container/heap"
	"fmt"

	"github.com/douban/gobeansdb/utils"
)

type mergeReader struct {
	r    *hintFileReader
	curr *HintItem
}

type mergeWriter struct {
	w   *hintFileWriter
	buf []*HintItem // always contain items with same khash and diff keys
	ct  *CollisionTable
	num int
}

func newMergeWriter(w *hintFileWriter, ct *CollisionTable) *mergeWriter {
	mw := new(mergeWriter)
	mw.w = w
	mw.buf = make([]*HintItem, 1000)
	mw.ct = ct
	return mw
}

func (mw *mergeWriter) write(it *HintItem) {
	if mw.num == 0 { // the first
		mw.buf[0] = it
		mw.num = 1
		return
	}
	last := mw.buf[mw.num-1]
	if last.Keyhash != it.Keyhash {
		mw.flush()
		mw.num = 1
		mw.buf[0] = it
	} else {
		if last.Key != it.Key {
			mw.num += 1
			if mw.num > len(mw.buf) {
				newbuf := make([]*HintItem, len(mw.buf)*2)
				copy(newbuf, mw.buf)
				mw.buf = newbuf
			}
		}
		mw.buf[mw.num-1] = it
	}
}

func (mw *mergeWriter) flush() {
	if mw.num > 1 {
		for i := 0; i < mw.num; i++ {
			mw.ct.compareAndSet(mw.buf[i], "merge")
		}
	}
	if mw.w != nil {
		for i := 0; i < mw.num; i++ {
			mw.w.writeItem(mw.buf[i])
		}
	}
}

type mergeHeap []*mergeReader

func (h mergeHeap) Len() int      { return len(h) }
func (h mergeHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h mergeHeap) Less(i, j int) bool {
	a := h[i].curr
	b := h[j].curr
	if a.Keyhash != b.Keyhash {
		return a.Keyhash < b.Keyhash
	} else {
		if a.Key != b.Key {
			return a.Key < b.Key
		}
	}
	return a.Pos.CmpKey() < b.Pos.CmpKey()
}

func (h *mergeHeap) Push(x interface{}) {
	*h = append(*h, x.(*mergeReader))
}

func (h *mergeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func merge(src []*hintFileReader, dst string, ct *CollisionTable, hintState *int, forGC bool) (idx *hintFileIndex, err error) {
	n := len(src)
	datasize := uint32(0)
	hp := make([]*mergeReader, n)
	for i := 0; i < n; i++ {
		err := src[i].open()
		if err != nil {
			logger.Errorf("%s", err.Error())
			return nil, err
		}
		hp[i] = &mergeReader{src[i], nil}
		hp[i].curr, err = src[i].next()
		hp[i].curr.Pos.ChunkID = src[i].chunkID
		if err != nil {
			logger.Errorf("%s", err.Error())
			return nil, err
		}
		if src[i].datasize > datasize {
			datasize = src[i].datasize
		}
	}
	var w *hintFileWriter
	if !Conf.NoMerged && !forGC {
		w, err = newHintFileWriter(dst, datasize, 1<<20)
		if err != nil {
			logger.Errorf("%s", err.Error())
			w = nil
		}
	}

	mw := newMergeWriter(w, ct)
	h := mergeHeap(hp)
	heap.Init(&h)
	for len(h) > 0 {
		if *hintState&HintStateGC != 0 && !forGC {
			err = fmt.Errorf("aborted by gc")
			break
		}
		mr := heap.Pop(&h).(*mergeReader)
		mw.write(mr.curr)
		mr.curr, err = mr.r.next()
		if err != nil {
			logger.Errorf("%s", err.Error())
			break
		}
		if mr.curr != nil {
			mr.curr.Pos.ChunkID = mr.r.chunkID
			heap.Push(&h, mr)
		}
	}
	for _, mr := range hp {
		mr.r.close()
	}
	mw.flush()
	if mw.w != nil {
		mw.w.close()
		idx = &hintFileIndex{mw.w.index.toIndex(), dst, w.hintFileMeta}
	}
	if err != nil {
		utils.Remove(dst)
		return nil, err
	}
	return
}
