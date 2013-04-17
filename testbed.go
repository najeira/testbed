package testbed

import (
	"appengine_internal"
	basepb "appengine_internal/base"
	"appengine_internal/remote_api"
	"bufio"
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Testbed struct {
	pipe     *exec.Cmd
	apiRead  *bufio.Reader
	apiWrite *bufio.Writer
	apiErr   *bufio.Reader
	mu       sync.Mutex
	cmd      string
	args     []string
}

func NewTestbed(cmd, py string, arg ...string) *Testbed {
	f, err := os.Open(py)
	if err != nil {
		panic(err)
	}
	f.Close()

	carg := []string{py}
	for _, a := range arg {
		carg = append(carg, a)
	}

	t := &Testbed{}
	t.cmd = cmd
	t.args = carg
	return t
}

func (t *Testbed) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.pipe != nil {
		return
	}

	t.pipe = exec.Command(t.cmd, t.args...)

	w, err := t.pipe.StdinPipe()
	if err != nil {
		panic(err)
	}
	t.apiWrite = bufio.NewWriter(w)

	r, err := t.pipe.StdoutPipe()
	if err != nil {
		panic(err)
	}
	t.apiRead = bufio.NewReader(r)

	r, err = t.pipe.StderrPipe()
	if err != nil {
		panic(err)
	}
	t.apiErr = bufio.NewReader(r)

	if err := t.pipe.Start(); err != nil {
		panic(err)
	}
}

func (t *Testbed) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pipe == nil {
		return
	}
	t.mu.Unlock()
	t.quit()
	t.mu.Lock()
	donec := make(chan error, 1) // non block chan
	go func() {
		donec <- t.pipe.Wait()
	}()
	select {
	case <-time.After(3 * time.Second):
		t.pipe.Process.Kill()
	case <-donec:
		// ok
	}
	t.pipe = nil
}

func (t *Testbed) quit() error {
	if err := t.writeString("#quit#"); err != nil {
		return err
	}
	return nil
}

func (t *Testbed) Reset() error {
	if err := t.writeString("#reset#"); err != nil {
		return err
	}
	return nil
}

func (t *Testbed) writeString(msg string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, err := t.apiWrite.WriteString(msg); err != nil {
		return err
	}
	if _, err := t.apiWrite.WriteRune('\n'); err != nil {
		return err
	}
	return t.apiWrite.Flush()
}

func (t *Testbed) Run(f func()) {
	t.Start()
	defer t.Close()
	f()
}

// read and write speak a custom protocol with the appserver. Specifically, an
// ASCII header followed by an encoded protocol buffer. The header is the
// length of the protocol buffer, in decimal, followed by a new line character.
// For example: "53\n".

// read reads a protocol buffer from the socketAPI socket.
func read(r *bufio.Reader, pb proto.Message) error {
	s, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	s = strings.TrimRight(s, "\r\n")
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	return proto.Unmarshal(b, pb)
}

// write writes a protocol buffer to the socketAPI socket.
func write(w *bufio.Writer, pb proto.Message) error {
	b, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	buf.WriteString(strconv.Itoa(len(b)))
	buf.WriteString("\n")
	buf.Write(b)

	body := base64.StdEncoding.EncodeToString(buf.Bytes())

	w.WriteString(body)
	w.WriteRune('\n')
	return w.Flush()
}

func (t *Testbed) call(service, method string, data []byte, requestID string) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	req := &remote_api.Request{
		ServiceName: &service,
		Method:      &method,
		Request:     data,
		RequestId:   &requestID,
	}
	if err := write(t.apiWrite, req); err != nil {
		return nil, err
	}
	res := &remote_api.Response{}
	if err := read(t.apiRead, res); err != nil {
		return nil, err
	}
	if ae := res.ApplicationError; ae != nil {
		// All Remote API application errors are API-level failures.
		return nil, &appengine_internal.APIError{Service: service, Detail: *ae.Detail, Code: *ae.Code}
	}
	return res.Response, nil
}

// context represents the context of an in-flight HTTP request.
// It implements the appengine.Context interface.
// Really, this is a copy & paste from appengine_internal, no changes.
// I wanted it here just to play and see how it works.
type context struct {
	req *http.Request
	t   *Testbed
}

func (t *Testbed) NewContext(req *http.Request) *context {
	return &context{req, t}
}

func (c *context) Call(service, method string, in, out appengine_internal.ProtoMessage, opts *appengine_internal.CallOptions) error {
	if service == "__go__" {
		if method == "GetNamespace" {
			out.(*basepb.StringProto).Value = proto.String(c.req.Header.Get("X-AppEngine-Current-Namespace"))
			return nil
		}
		if method == "GetDefaultNamespace" {
			out.(*basepb.StringProto).Value = proto.String(c.req.Header.Get("X-AppEngine-Default-Namespace"))
			return nil
		}
	}
	data, err := proto.Marshal(in)
	if err != nil {
		return err
	}
	requestID := c.req.Header.Get("X-Appengine-Internal-Request-Id")
	res, err := c.t.call(service, method, data, requestID)
	if err != nil {
		return err
	}
	return proto.Unmarshal(res, out)
}

func (c *context) Request() interface{} {
	return c.req
}

func (c *context) logf(level, format string, args ...interface{}) {
	log.Printf(level+": "+format, args...)
}

func (c *context) Debugf(format string, args ...interface{})    { c.logf("DEBUG", format, args...) }
func (c *context) Infof(format string, args ...interface{})     { c.logf("INFO", format, args...) }
func (c *context) Warningf(format string, args ...interface{})  { c.logf("WARNING", format, args...) }
func (c *context) Errorf(format string, args ...interface{})    { c.logf("ERROR", format, args...) }
func (c *context) Criticalf(format string, args ...interface{}) { c.logf("CRITICAL", format, args...) }

// FullyQualifiedAppID returns the fully-qualified application ID.
// This may contain a partition prefix (e.g. "s~" for High Replication apps),
// or a domain prefix (e.g. "example.com:").
func (c *context) FullyQualifiedAppID() string {
	return "testbed-test"
}
