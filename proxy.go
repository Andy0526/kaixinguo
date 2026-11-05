package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kadaliao/kaixinguo/internal/cenc"
)

// proxy serves the player page and the episodes to the web view, from
// 127.0.0.1. Episodes are decrypted as they stream from the CDN, a range
// at a time, so the player starts at once and seeks anywhere.
type proxy struct {
	host    string // 127.0.0.1:port
	base    string // http://127.0.0.1:port/<secret>
	client  *http.Client
	onEvent func(url.Values)

	mu      sync.Mutex
	streams map[string]*stream
	order   []string
}

// stream is one episode's file on the CDN.
type stream struct {
	urls []string
	key  []byte // nil for a clear file

	mu     sync.Mutex
	ready  bool
	size   int64
	layout *cenc.Layout
	good   int // the URL that answered
}

func newProxy(onEvent func(url.Values)) (*proxy, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	p := &proxy{
		host:    ln.Addr().String(),
		client:  &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 20 * time.Second, MaxIdleConnsPerHost: 8}},
		onEvent: onEvent,
		streams: map[string]*stream{},
	}
	p.base = "http://" + p.host + "/" + randomHex(12)
	srv := &http.Server{Handler: p, ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)
	return p, nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (p *proxy) playerURL() string { return p.base + "/player" }

// add makes the file at urls (a CDN URL and its backup) playable, and
// returns the URL the web view plays it from.
func (p *proxy) add(urls []string, key []byte) string {
	token := randomHex(8)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streams[token] = &stream{urls: urls, key: key}
	p.order = append(p.order, token)
	// Keep the last few: the one playing, and one the player may still read.
	for len(p.order) > 4 {
		delete(p.streams, p.order[0])
		p.order = p.order[1:]
	}
	return p.base + "/v/" + token + ".mp4"
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Host != p.host {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	prefix := strings.TrimPrefix(p.base, "http://"+p.host)
	rest, ok := strings.CutPrefix(r.URL.Path, prefix+"/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch {
	case rest == "player":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		io.WriteString(w, playerHTML)
	case rest == "event":
		if p.onEvent != nil {
			p.onEvent(r.URL.Query())
		}
		w.WriteHeader(http.StatusNoContent)
	case strings.HasPrefix(rest, "v/"):
		token := strings.TrimSuffix(strings.TrimPrefix(rest, "v/"), ".mp4")
		p.mu.Lock()
		s := p.streams[token]
		p.mu.Unlock()
		if s == nil {
			http.NotFound(w, r)
			return
		}
		p.serveStream(w, r, s)
	default:
		http.NotFound(w, r)
	}
}

func (p *proxy) serveStream(w http.ResponseWriter, r *http.Request, s *stream) {
	ctx := r.Context()
	if err := p.prepare(ctx, s); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	start, end, partial, ok := parseRange(r.Header.Get("Range"), s.size)
	if !ok {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", s.size))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	var body io.ReadCloser
	if r.Method != http.MethodHead {
		var err error
		if body, err = p.open(ctx, s, start, end); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer body.Close()
	}
	h := w.Header()
	h.Set("Content-Type", "video/mp4")
	h.Set("Accept-Ranges", "bytes")
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Length", strconv.FormatInt(end-start+1, 10))
	if partial {
		h.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, s.size))
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	if body == nil {
		return
	}
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 64<<10)
	for pos := start; pos <= end; {
		n, err := body.Read(buf[:min(int64(len(buf)), end-pos+1)])
		if n > 0 {
			chunk := buf[:n]
			if s.layout != nil {
				s.layout.Decrypt(chunk, pos)
			}
			if _, werr := w.Write(chunk); werr != nil {
				return // the player let go of this range
			}
			if flusher != nil {
				flusher.Flush()
			}
			pos += int64(n)
		}
		if err != nil {
			return
		}
	}
}

// prepare learns the file's size and, for an encrypted one, its layout.
func (p *proxy) prepare(ctx context.Context, s *stream) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ready {
		return nil
	}
	var lastErr error
	for i, u := range s.urls {
		head, total, err := p.fetch(ctx, u, 0, 64<<10-1)
		if err != nil {
			lastErr = err
			continue
		}
		s.size, s.good = total, i
		if s.key != nil {
			read := func(off, n int64) ([]byte, error) {
				if off+n <= int64(len(head)) {
					return head[off : off+n], nil
				}
				b, _, err := p.fetch(ctx, u, off, off+n-1)
				return b, err
			}
			l, err := cenc.Open(read, total, s.key)
			if err != nil {
				return err
			}
			s.layout = l
		}
		s.ready = true
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("no stream URL")
	}
	return lastErr
}

// fetch reads bytes a to b of u, and the file's size.
func (p *proxy) fetch(ctx context.Context, u string, a, b int64) ([]byte, int64, error) {
	resp, err := p.get(ctx, u, a, b)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	total := resp.ContentLength
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		if i := strings.LastIndexByte(cr, '/'); i >= 0 {
			total, _ = strconv.ParseInt(cr[i+1:], 10, 64)
		}
	}
	if resp.StatusCode == http.StatusOK && a > 0 {
		if _, err := io.CopyN(io.Discard, resp.Body, a); err != nil {
			return nil, 0, err
		}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, b-a+1))
	if err != nil {
		return nil, 0, err
	}
	if total <= 0 {
		return nil, 0, errors.New("the CDN did not tell the file's size")
	}
	return data, total, nil
}

// open streams bytes a to b of the stream, from the URL that answered,
// else the other.
func (p *proxy) open(ctx context.Context, s *stream, a, b int64) (io.ReadCloser, error) {
	order := []int{s.good}
	for i := range s.urls {
		if i != s.good {
			order = append(order, i)
		}
	}
	var lastErr error
	for _, i := range order {
		resp, err := p.get(ctx, s.urls[i], a, b)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusOK && a > 0 {
			if _, err := io.CopyN(io.Discard, resp.Body, a); err != nil {
				resp.Body.Close()
				lastErr = err
				continue
			}
		}
		return resp.Body, nil
	}
	return nil, lastErr
}

func (p *proxy) get(ctx context.Context, u string, a, b int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", a, b))
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("CDN answered %s", resp.Status)
	}
	return resp, nil
}

// parseRange reads a Range header of one range against a file of size
// bytes; partial is false for no header.
func parseRange(h string, size int64) (start, end int64, partial, ok bool) {
	if h == "" {
		return 0, size - 1, false, size > 0
	}
	spec, found := strings.CutPrefix(h, "bytes=")
	if !found || strings.Contains(spec, ",") {
		return 0, 0, false, false
	}
	first, last, found := strings.Cut(strings.TrimSpace(spec), "-")
	if !found {
		return 0, 0, false, false
	}
	if first == "" { // the last n bytes
		n, err := strconv.ParseInt(last, 10, 64)
		if err != nil || n <= 0 {
			return 0, 0, false, false
		}
		return max(0, size-n), size - 1, true, size > 0
	}
	start, err := strconv.ParseInt(first, 10, 64)
	if err != nil || start < 0 || start >= size {
		return 0, 0, false, false
	}
	end = size - 1
	if last != "" {
		if end, err = strconv.ParseInt(last, 10, 64); err != nil || end < start {
			return 0, 0, false, false
		}
		end = min(end, size-1)
	}
	return start, end, true, true
}

// playerHTML is the page of the web view: a video and the calls the app
// makes to it. It tells the app what the video does through /event.
const playerHTML = `<!doctype html>
<html><head><meta charset="utf-8">
<style>
html,body{margin:0;height:100%;background:#000;overflow:hidden;-webkit-user-select:none;cursor:default}
video{position:absolute;inset:0;width:100%;height:100%;object-fit:contain;background:#000;outline:none}
</style></head>
<body><video id="v" controls playsinline preload="auto"></video>
<script>
const v = document.getElementById('v');
let ep = '', last = 0;
function send(type, extra) {
  fetch('event?type=' + type + '&ep=' + encodeURIComponent(ep) + (extra || '')).catch(() => {});
}
window.kx = {
  play(src, id, start) {
    ep = id; last = 0;
    v.src = src;
    if (start > 0) v.addEventListener('loadedmetadata', () => { v.currentTime = start }, {once: true});
    v.play().catch(e => { if (e.name !== 'AbortError') send('error', '&msg=' + encodeURIComponent(e.message)) });
  },
  stop() { v.pause(); v.removeAttribute('src'); v.load(); },
  pause() { v.pause() },
  toggle() { v.paused ? v.play() : v.pause() },
  seek(d) { v.currentTime = Math.max(0, v.currentTime + d) },
};
v.addEventListener('ended', () => send('ended'));
v.addEventListener('playing', () => send('playing'));
v.addEventListener('waiting', () => send('waiting'));
v.addEventListener('error', () => send('error', '&msg=' + (v.error ? v.error.code : '')));
v.addEventListener('timeupdate', () => {
  const now = Date.now();
  if (now - last < 3000) return;
  last = now;
  send('progress', '&t=' + v.currentTime.toFixed(1) + '&d=' + (v.duration || 0).toFixed(1));
});
v.addEventListener('pause', () => {
  if (v.currentTime > 0) send('progress', '&t=' + v.currentTime.toFixed(1) + '&d=' + (v.duration || 0).toFixed(1));
});
document.addEventListener('keydown', e => {
  if (e.key === ' ') { e.preventDefault(); kx.toggle() }
  else if (e.key === 'ArrowLeft') { e.preventDefault(); kx.seek(-5) }
  else if (e.key === 'ArrowRight') { e.preventDefault(); kx.seek(5) }
  else if (e.key === 'ArrowDown') { e.preventDefault(); send('next') }
  else if (e.key === 'ArrowUp') { e.preventDefault(); send('prev') }
});
</script></body></html>`
