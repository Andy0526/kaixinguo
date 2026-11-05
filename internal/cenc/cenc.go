// Package cenc turns a progressive MP4 encrypted with the 'cenc' scheme
// (AES-CTR, ISO/IEC 23001-7) into a clear one, a byte range at a time.
//
// Decryption never changes a byte's position: samples are decrypted in
// place and the boxes that mark the file as protected are renamed to boxes
// of the same size ('encv' → 'hvc1', 'sinf' → 'free', …). So any range of
// the clear file is the same range of the encrypted file, decrypted, and a
// player can seek through a stream that is decrypted as it downloads.
package cenc

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

// Fetch reads n bytes of the file at off.
type Fetch func(off, n int64) ([]byte, error)

// Layout is what Decrypt needs to know about one file: where its samples
// are, their IVs, and the patched header.
type Layout struct {
	block   cipher.Block
	moovOff int64
	moov    []byte   // the patched moov box, written over the original
	samples []sample // protected samples, sorted by offset
}

type sample struct {
	off  int64
	size uint32
	iv   [16]byte
	subs []subsample // nil: the whole sample is encrypted
}

type subsample struct {
	clear uint32
	enc   uint32
}

// Open reads the moov box of a file of total bytes through fetch and
// returns its layout for the 16-byte content key.
func Open(fetch Fetch, total int64, key []byte) (*Layout, error) {
	moovOff, moov, err := findMoov(fetch, total)
	if err != nil {
		return nil, err
	}
	return Parse(moovOff, moov, key, fetch)
}

// findMoov walks the top-level boxes until it finds moov, which faststart
// files keep before mdat and others after it.
func findMoov(fetch Fetch, total int64) (int64, []byte, error) {
	const probe = 64 << 10
	head, err := fetch(0, min(probe, total))
	if err != nil {
		return 0, nil, err
	}
	var off int64
	for off+8 <= total {
		var hdr []byte
		if off+16 <= int64(len(head)) {
			hdr = head[off : off+16]
		} else {
			if hdr, err = fetch(off, min(16, total-off)); err != nil {
				return 0, nil, err
			}
		}
		size := int64(binary.BigEndian.Uint32(hdr))
		typ := string(hdr[4:8])
		if size == 1 && len(hdr) >= 16 {
			size = int64(binary.BigEndian.Uint64(hdr[8:]))
		} else if size == 0 {
			size = total - off
		}
		if size < 8 || off+size > total {
			return 0, nil, fmt.Errorf("cenc: bad box %q at %d", typ, off)
		}
		if typ == "moov" {
			if off+size <= int64(len(head)) {
				return off, head[off : off+size], nil
			}
			moov, err := fetch(off, size)
			return off, moov, err
		}
		off += size
	}
	return 0, nil, errors.New("cenc: no moov box")
}

// Parse builds the layout of a file whose moov box, at moovOff, is moov.
// fetch reads auxiliary data that saio places outside moov; it may be nil
// when the file keeps its IVs in senc boxes.
func Parse(moovOff int64, moov []byte, key []byte, fetch Fetch) (*Layout, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	l := &Layout{block: block, moovOff: moovOff, moov: append([]byte(nil), moov...)}
	root, ok := parseBox(l.moov, 0, len(l.moov))
	if !ok || root.typ != "moov" {
		return nil, errors.New("cenc: not a moov box")
	}
	for _, b := range root.children() {
		switch b.typ {
		case "trak":
			if err := l.parseTrak(b, fetch); err != nil {
				return nil, err
			}
		case "pssh":
			b.rename("free")
		}
	}
	sort.Slice(l.samples, func(i, j int) bool { return l.samples[i].off < l.samples[j].off })
	return l, nil
}

// Protected reports whether the file had any encrypted samples.
func (l *Layout) Protected() bool { return len(l.samples) > 0 }

// Decrypt turns buf, the bytes of the file at off, into the clear file's.
func (l *Layout) Decrypt(buf []byte, off int64) {
	end := off + int64(len(buf))
	// The patched header.
	if mEnd := l.moovOff + int64(len(l.moov)); off < mEnd && end > l.moovOff {
		a, b := max(off, l.moovOff), min(end, mEnd)
		copy(buf[a-off:b-off], l.moov[a-l.moovOff:b-l.moovOff])
	}
	// The samples.
	i := sort.Search(len(l.samples), func(i int) bool {
		s := &l.samples[i]
		return s.off+int64(s.size) > off
	})
	for ; i < len(l.samples) && l.samples[i].off < end; i++ {
		s := &l.samples[i]
		if s.subs == nil {
			a, b := max(off, s.off), min(end, s.off+int64(s.size))
			l.xor(buf[a-off:b-off], &s.iv, a-s.off)
			continue
		}
		// Subsamples: the encrypted parts make one CTR stream.
		pos, stream := s.off, int64(0)
		for _, sub := range s.subs {
			pos += int64(sub.clear)
			encEnd := pos + int64(sub.enc)
			if a, b := max(off, pos), min(end, encEnd); a < b {
				l.xor(buf[a-off:b-off], &s.iv, stream+(a-pos))
			}
			stream += int64(sub.enc)
			pos = encEnd
		}
	}
}

// xor applies the keystream of iv, from byte at of its stream, to dst.
// The counter is the IV's low 64 bits, as CENC and ffmpeg count blocks.
func (l *Layout) xor(dst []byte, iv *[16]byte, at int64) {
	blk := uint64(at / 16)
	skip := int(at % 16)
	var ctr, ks [16]byte
	hi := binary.BigEndian.Uint64(iv[:8])
	lo := binary.BigEndian.Uint64(iv[8:])
	for len(dst) > 0 {
		binary.BigEndian.PutUint64(ctr[:8], hi)
		binary.BigEndian.PutUint64(ctr[8:], lo+blk)
		l.block.Encrypt(ks[:], ctr[:])
		n := min(len(dst), 16-skip)
		for k := 0; k < n; k++ {
			dst[k] ^= ks[skip+k]
		}
		dst = dst[n:]
		skip = 0
		blk++
	}
}

// track is what one trak says about its protection.
type track struct {
	ivSize      int
	constIV     []byte
	protected   bool
	sizes       []uint32
	offsets     []int64
	ivs         [][16]byte
	subs        [][]subsample
	senc        bool
	saizDefault int
	saizSizes   []uint8
	saioOffsets []int64
}

func (l *Layout) parseTrak(trak box, fetch Fetch) error {
	stbl, ok := trak.find("mdia", "minf", "stbl")
	if !ok {
		return nil
	}
	stsd, ok := stbl.find("stsd")
	if !ok || len(stsd.body) < 8 {
		return nil
	}
	entry, ok := parseBox(stsd.buf, stsd.bodyAt+8, stsd.end)
	if !ok {
		return nil
	}
	var start int
	switch entry.typ {
	case "encv":
		start = 78
	case "enca":
		start = 28
		if len(entry.body) >= 10 {
			switch binary.BigEndian.Uint16(entry.body[8:]) { // QuickTime sound versions
			case 1:
				start += 16
			case 2:
				start += 36
			}
		}
	default:
		return nil // a clear track
	}
	t := &track{}
	var format string
	for _, b := range entry.childrenFrom(start) {
		if b.typ != "sinf" {
			continue
		}
		if frma, ok := b.find("frma"); ok && len(frma.body) >= 4 {
			format = string(frma.body[:4])
		}
		if schm, ok := b.find("schm"); ok && len(schm.body) >= 8 {
			if s := string(schm.body[4:8]); s != "cenc" {
				return fmt.Errorf("cenc: scheme %q is not supported", s)
			}
		}
		tenc, ok := b.find("schi", "tenc")
		if !ok || len(tenc.body) < 24 {
			return errors.New("cenc: no tenc box")
		}
		t.protected = tenc.body[6] != 0
		t.ivSize = int(tenc.body[7])
		if t.protected && t.ivSize == 0 && len(tenc.body) >= 25 {
			n := int(tenc.body[24])
			if len(tenc.body) >= 25+n {
				t.constIV = tenc.body[25 : 25+n]
			}
		}
		b.rename("free")
	}
	if format == "" {
		return errors.New("cenc: no original format")
	}
	entry.rename(format)
	if !t.protected {
		return nil
	}

	var stsc []uint32
	for _, b := range stbl.children() {
		body := b.body
		switch b.typ {
		case "stsz":
			if len(body) < 12 {
				return errors.New("cenc: bad stsz")
			}
			fixed, n := binary.BigEndian.Uint32(body[4:]), int(binary.BigEndian.Uint32(body[8:]))
			t.sizes = make([]uint32, n)
			for k := range t.sizes {
				if fixed != 0 {
					t.sizes[k] = fixed
				} else if 12+4*k+4 <= len(body) {
					t.sizes[k] = binary.BigEndian.Uint32(body[12+4*k:])
				}
			}
		case "stsc":
			n := int(binary.BigEndian.Uint32(body[4:]))
			for k := 0; k < n && 8+12*k+12 <= len(body); k++ {
				e := body[8+12*k:]
				stsc = append(stsc, binary.BigEndian.Uint32(e), binary.BigEndian.Uint32(e[4:]))
			}
		case "stco", "co64":
			n := int(binary.BigEndian.Uint32(body[4:]))
			w := 4
			if b.typ == "co64" {
				w = 8
			}
			t.offsets = make([]int64, 0, n)
			for k := 0; k < n && 8+w*k+w <= len(body); k++ {
				if w == 4 {
					t.offsets = append(t.offsets, int64(binary.BigEndian.Uint32(body[8+4*k:])))
				} else {
					t.offsets = append(t.offsets, int64(binary.BigEndian.Uint64(body[8+8*k:])))
				}
			}
		case "senc":
			if err := t.parseSenc(body); err != nil {
				return err
			}
			b.rename("free")
		case "saiz":
			p := 4
			if body[3]&1 != 0 {
				p += 8
			}
			if len(body) < p+5 {
				return errors.New("cenc: bad saiz")
			}
			t.saizDefault = int(body[p])
			n := int(binary.BigEndian.Uint32(body[p+1:]))
			if t.saizDefault == 0 && len(body) >= p+5+n {
				t.saizSizes = body[p+5 : p+5+n]
			}
			b.rename("free")
		case "saio":
			p := 4
			if body[3]&1 != 0 {
				p += 8
			}
			n := int(binary.BigEndian.Uint32(body[p:]))
			w := 4
			if body[0] == 1 {
				w = 8
			}
			for k := 0; k < n && p+4+w*k+w <= len(body); k++ {
				e := body[p+4+w*k:]
				if w == 4 {
					t.saioOffsets = append(t.saioOffsets, int64(binary.BigEndian.Uint32(e)))
				} else {
					t.saioOffsets = append(t.saioOffsets, int64(binary.BigEndian.Uint64(e)))
				}
			}
			b.rename("free")
		}
	}
	if !t.senc {
		if err := t.readAux(l, fetch); err != nil {
			return err
		}
	}
	return l.addSamples(t, stsc)
}

func (t *track) parseSenc(body []byte) error {
	if len(body) < 8 {
		return errors.New("cenc: bad senc")
	}
	t.senc = true
	flags := binary.BigEndian.Uint32(body) & 0xffffff
	n := int(binary.BigEndian.Uint32(body[4:]))
	_, err := t.readInfo(body[8:], n, flags&2 != 0, nil)
	return err
}

// readInfo reads n samples' IVs, and their subsamples when subs, from b.
// sizes, when given, is each sample's info size, which says whether it
// has subsamples.
func (t *track) readInfo(b []byte, n int, subs bool, sizes func(int) int) (int, error) {
	p := 0
	t.ivs = make([][16]byte, n)
	t.subs = make([][]subsample, n)
	for k := 0; k < n; k++ {
		start := p
		if t.ivSize > 0 {
			if p+t.ivSize > len(b) {
				return p, errors.New("cenc: sample info is short")
			}
			copy(t.ivs[k][:], b[p:p+t.ivSize])
			p += t.ivSize
		} else {
			copy(t.ivs[k][:], t.constIV)
		}
		has := subs
		if sizes != nil {
			has = sizes(k) > t.ivSize
		}
		if has {
			if p+2 > len(b) {
				return p, errors.New("cenc: subsample count is short")
			}
			m := int(binary.BigEndian.Uint16(b[p:]))
			p += 2
			if p+6*m > len(b) {
				return p, errors.New("cenc: subsamples are short")
			}
			t.subs[k] = make([]subsample, m)
			for j := range m {
				t.subs[k][j] = subsample{uint32(binary.BigEndian.Uint16(b[p:])), binary.BigEndian.Uint32(b[p+2:])}
				p += 6
			}
		}
		if sizes != nil {
			p = start + sizes(k)
		}
	}
	return p, nil
}

// readAux reads the sample info that saiz sizes and saio places, inside
// moov or through fetch.
func (t *track) readAux(l *Layout, fetch Fetch) error {
	if len(t.saioOffsets) != 1 {
		return errors.New("cenc: no sample encryption info")
	}
	n := len(t.sizes)
	size := func(k int) int {
		if t.saizDefault != 0 {
			return t.saizDefault
		}
		if k < len(t.saizSizes) {
			return int(t.saizSizes[k])
		}
		return t.ivSize
	}
	total := 0
	for k := range n {
		total += size(k)
	}
	off := t.saioOffsets[0]
	var b []byte
	if off >= l.moovOff && off+int64(total) <= l.moovOff+int64(len(l.moov)) {
		b = l.moov[off-l.moovOff : off-l.moovOff+int64(total)]
	} else if fetch != nil {
		var err error
		if b, err = fetch(off, int64(total)); err != nil {
			return err
		}
	} else {
		return errors.New("cenc: sample info is outside moov")
	}
	_, err := t.readInfo(b, n, false, size)
	return err
}

// addSamples places the track's samples in the file from its chunks.
func (l *Layout) addSamples(t *track, stsc []uint32) error {
	if len(stsc) == 0 || len(t.offsets) == 0 {
		return errors.New("cenc: no chunks")
	}
	if len(t.ivs) < len(t.sizes) {
		return fmt.Errorf("cenc: %d IVs for %d samples", len(t.ivs), len(t.sizes))
	}
	k := 0
	for e := 0; e < len(stsc); e += 2 {
		first, per := int(stsc[e])-1, int(stsc[e+1])
		last := len(t.offsets)
		if e+2 < len(stsc) {
			last = int(stsc[e+2]) - 1
		}
		for chunk := first; chunk < last && chunk < len(t.offsets); chunk++ {
			off := t.offsets[chunk]
			for j := 0; j < per && k < len(t.sizes); j++ {
				l.samples = append(l.samples, sample{off: off, size: t.sizes[k], iv: t.ivs[k], subs: t.subs[k]})
				off += int64(t.sizes[k])
				k++
			}
		}
	}
	if k != len(t.sizes) {
		return fmt.Errorf("cenc: chunks hold %d of %d samples", k, len(t.sizes))
	}
	return nil
}

// box is an MP4 box inside a buffer that it can rename in place.
type box struct {
	buf    []byte
	at     int // the header
	bodyAt int
	end    int
	typ    string
	body   []byte
}

// parseBox reads the box at at, which must end by limit.
func parseBox(buf []byte, at, limit int) (box, bool) {
	if at+8 > limit {
		return box{}, false
	}
	size := int(binary.BigEndian.Uint32(buf[at:]))
	hdr := 8
	if size == 1 {
		if at+16 > limit {
			return box{}, false
		}
		size = int(binary.BigEndian.Uint64(buf[at+8:]))
		hdr = 16
	} else if size == 0 {
		size = limit - at
	}
	if size < hdr || at+size > limit {
		return box{}, false
	}
	return box{buf: buf, at: at, bodyAt: at + hdr, end: at + size, typ: string(buf[at+4 : at+8]), body: buf[at+hdr : at+size]}, true
}

func (b box) children() []box { return b.childrenFrom(0) }

// childrenFrom lists the boxes in b's body from byte start of it on.
func (b box) childrenFrom(start int) []box {
	var out []box
	for at := b.bodyAt + start; at < b.end; {
		c, ok := parseBox(b.buf, at, b.end)
		if !ok {
			break
		}
		out = append(out, c)
		at = c.end
	}
	return out
}

// find follows a path of child box types.
func (b box) find(path ...string) (box, bool) {
	cur := b
	for _, typ := range path {
		found := false
		for _, c := range cur.children() {
			if c.typ == typ {
				cur, found = c, true
				break
			}
		}
		if !found {
			return box{}, false
		}
	}
	return cur, true
}

func (b box) rename(typ string) { copy(b.buf[b.at+4:b.at+8], typ) }
