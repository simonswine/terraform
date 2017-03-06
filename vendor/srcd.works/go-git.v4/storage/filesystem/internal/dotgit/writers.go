package dotgit

import (
	"fmt"
	"io"
	"sync/atomic"

	"srcd.works/go-git.v4/plumbing"
	"srcd.works/go-git.v4/plumbing/format/idxfile"
	"srcd.works/go-git.v4/plumbing/format/objfile"
	"srcd.works/go-git.v4/plumbing/format/packfile"

	"srcd.works/go-billy.v1"
)

// PackWriter is a io.Writer that generates the packfile index simultaneously,
// a packfile.Decoder is used with a file reader to read the file being written
// this operation is synchronized with the write operations.
// The packfile is written in a temp file, when Close is called this file
// is renamed/moved (depends on the Filesystem implementation) to the final
// location, if the PackWriter is not used, nothing is written
type PackWriter struct {
	Notify func(h plumbing.Hash, i idxfile.Idxfile)

	fs       billy.Filesystem
	fr, fw   billy.File
	synced   *syncedReader
	checksum plumbing.Hash
	index    idxfile.Idxfile
	result   chan error
}

func newPackWrite(fs billy.Filesystem) (*PackWriter, error) {
	fw, err := fs.TempFile(fs.Join(objectsPath, packPath), "tmp_pack_")
	if err != nil {
		return nil, err
	}

	fr, err := fs.Open(fw.Filename())
	if err != nil {
		return nil, err
	}

	writer := &PackWriter{
		fs:     fs,
		fw:     fw,
		fr:     fr,
		synced: newSyncedReader(fw, fr),
		result: make(chan error),
	}

	go writer.buildIndex()
	return writer, nil
}

func (w *PackWriter) buildIndex() {
	s := packfile.NewScanner(w.synced)
	d, err := packfile.NewDecoder(s, nil)
	if err != nil {
		w.result <- err
		return
	}

	checksum, err := d.Decode()
	if err != nil {
		w.result <- err
		return
	}

	w.checksum = checksum
	w.index.PackfileChecksum = checksum
	w.index.Version = idxfile.VersionSupported

	offsets := d.Offsets()
	for h, crc := range d.CRCs() {
		w.index.Add(h, uint64(offsets[h]), crc)
	}

	w.result <- err
}

// waitBuildIndex waits until buildIndex function finishes, this can terminate
// with a packfile.ErrEmptyPackfile, this means that nothing was written so we
// ignore the error
func (w *PackWriter) waitBuildIndex() error {
	err := <-w.result
	if err == packfile.ErrEmptyPackfile {
		return nil
	}

	return err
}

func (w *PackWriter) Write(p []byte) (int, error) {
	return w.synced.Write(p)
}

// Close closes all the file descriptors and save the final packfile, if nothing
// was written, the tempfiles are deleted without writing a packfile.
func (w *PackWriter) Close() error {
	defer func() {
		if w.Notify != nil {
			w.Notify(w.checksum, w.index)
		}

		close(w.result)
	}()

	if err := w.synced.Close(); err != nil {
		return err
	}

	if err := w.waitBuildIndex(); err != nil {
		return err
	}

	if err := w.fr.Close(); err != nil {
		return err
	}

	if err := w.fw.Close(); err != nil {
		return err
	}

	if len(w.index.Entries) == 0 {
		return w.clean()
	}

	return w.save()
}

func (w *PackWriter) clean() error {
	return w.fs.Remove(w.fw.Filename())
}

func (w *PackWriter) save() error {
	base := w.fs.Join(objectsPath, packPath, fmt.Sprintf("pack-%s", w.checksum))
	idx, err := w.fs.Create(fmt.Sprintf("%s.idx", base))
	if err != nil {
		return err
	}

	if err := w.encodeIdx(idx); err != nil {
		return err
	}

	if err := idx.Close(); err != nil {
		return err
	}

	return w.fs.Rename(w.fw.Filename(), fmt.Sprintf("%s.pack", base))
}

func (w *PackWriter) encodeIdx(writer io.Writer) error {
	e := idxfile.NewEncoder(writer)
	_, err := e.Encode(&w.index)
	return err
}

type syncedReader struct {
	w io.Writer
	r io.ReadSeeker

	blocked, done uint32
	written, read uint64
	news          chan bool
}

func newSyncedReader(w io.Writer, r io.ReadSeeker) *syncedReader {
	return &syncedReader{
		w:    w,
		r:    r,
		news: make(chan bool),
	}
}

func (s *syncedReader) Write(p []byte) (n int, err error) {
	defer func() {
		written := atomic.AddUint64(&s.written, uint64(n))
		read := atomic.LoadUint64(&s.read)
		if written > read {
			s.wake()
		}
	}()

	n, err = s.w.Write(p)
	return
}

func (s *syncedReader) Read(p []byte) (n int, err error) {
	defer func() { atomic.AddUint64(&s.read, uint64(n)) }()

	s.sleep()
	n, err = s.r.Read(p)
	if err == io.EOF && !s.isDone() {
		if n == 0 {
			return s.Read(p)
		}

		return n, nil
	}

	return
}

func (s *syncedReader) isDone() bool {
	return atomic.LoadUint32(&s.done) == 1
}

func (s *syncedReader) isBlocked() bool {
	return atomic.LoadUint32(&s.blocked) == 1
}

func (s *syncedReader) wake() {
	if s.isBlocked() {
		//	fmt.Println("wake")
		atomic.StoreUint32(&s.blocked, 0)
		s.news <- true
	}
}

func (s *syncedReader) sleep() {
	read := atomic.LoadUint64(&s.read)
	written := atomic.LoadUint64(&s.written)
	if read >= written {
		atomic.StoreUint32(&s.blocked, 1)
		//	fmt.Println("sleep", read, written)
		<-s.news
	}

}

func (s *syncedReader) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekCurrent {
		return s.r.Seek(offset, whence)
	}

	p, err := s.r.Seek(offset, whence)
	s.read = uint64(p)

	return p, err
}

func (s *syncedReader) Close() error {
	atomic.StoreUint32(&s.done, 1)
	close(s.news)
	return nil
}

type ObjectWriter struct {
	objfile.Writer
	fs billy.Filesystem
	f  billy.File
}

func newObjectWriter(fs billy.Filesystem) (*ObjectWriter, error) {
	f, err := fs.TempFile(fs.Join(objectsPath, packPath), "tmp_obj_")
	if err != nil {
		return nil, err
	}

	return &ObjectWriter{
		Writer: (*objfile.NewWriter(f)),
		fs:     fs,
		f:      f,
	}, nil
}

func (w *ObjectWriter) Close() error {
	if err := w.Writer.Close(); err != nil {
		return err
	}

	if err := w.f.Close(); err != nil {
		return err
	}

	return w.save()
}

func (w *ObjectWriter) save() error {
	hash := w.Hash().String()
	file := w.fs.Join(objectsPath, hash[0:2], hash[2:40])

	return w.fs.Rename(w.f.Filename(), file)
}
