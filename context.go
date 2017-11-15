// Copyright 2015-2017 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tp

import (
	"compress/gzip"
	"net/url"
	"reflect"
	"time"

	"github.com/henrylee2cn/goutil"

	"github.com/henrylee2cn/teleport/socket"
)

type (
	// PushCtx push handler context.
	// For example:
	//  type HomePush struct{ PushCtx }
	PushCtx interface {
		Uri() string
		Path() string
		RawQuery() string
		Query() url.Values
		Public() goutil.Map
		PublicLen() int
		Ip() string
		Peer() *Peer
		Session() Session
	}
	UnknownPushCtx interface {
		PushCtx
		InputHeader() *socket.Header
		InputBodyBytes() []byte
		Bind(v interface{}) (codecName string, err error)
	}
	// PullCtx request handler context.
	// For example:
	//  type HomePull struct{ PullCtx }
	PullCtx interface {
		PushCtx
		SetBodyCodec(string)
	}
	UnknownPullCtx interface {
		UnknownPushCtx
		SetBodyCodec(string)
	}
	// WriteCtx for writing packet.
	WriteCtx interface {
		Output() *socket.Packet
		Public() goutil.Map
		PublicLen() int
		Ip() string
		Peer() *Peer
		Session() Session
	}
	// ReadCtx for reading packet.
	ReadCtx interface {
		Input() *socket.Packet
		Public() goutil.Map
		PublicLen() int
		Ip() string
		Peer() *Peer
		Session() Session
	}
	// readHandleCtx the underlying common instance of PullCtx and PushCtx.
	readHandleCtx struct {
		sess            *session
		input           *socket.Packet
		output          *socket.Packet
		apiType         *Handler
		arg             reflect.Value
		pullCmd         *PullCmd
		uri             *url.URL
		query           url.Values
		public          goutil.Map
		start           time.Time
		cost            time.Duration
		pluginContainer PluginContainer
		next            *readHandleCtx
	}
)

var (
	_ PullCtx        = new(readHandleCtx)
	_ PushCtx        = new(readHandleCtx)
	_ WriteCtx       = new(readHandleCtx)
	_ ReadCtx        = new(readHandleCtx)
	_ UnknownPullCtx = new(readHandleCtx)
	_ UnknownPushCtx = new(readHandleCtx)
)

var (
	emptyValue  = reflect.Value{}
	emptyMethod = reflect.Method{}
)

// newReadHandleCtx creates a readHandleCtx for one request/response or push.
func newReadHandleCtx() *readHandleCtx {
	c := new(readHandleCtx)
	c.input = socket.NewPacket(c.binding)
	c.output = socket.NewPacket(nil)
	return c
}

func (c *readHandleCtx) reInit(s *session) {
	c.sess = s
	count := s.socket.PublicLen()
	c.public = goutil.RwMap(count)
	if count > 0 {
		s.socket.Public().Range(func(key, value interface{}) bool {
			c.public.Store(key, value)
			return true
		})
	}
}

func (c *readHandleCtx) clean() {
	c.sess = nil
	c.apiType = nil
	c.arg = emptyValue
	c.pullCmd = nil
	c.public = nil
	c.uri = nil
	c.query = nil
	c.cost = 0
	c.pluginContainer = nil
	c.input.Reset(c.binding)
	c.output.Reset(nil)
}

// Peer returns the peer.
func (c *readHandleCtx) Peer() *Peer {
	return c.sess.peer
}

// Session returns the session.
func (c *readHandleCtx) Session() Session {
	return c.sess
}

// Input returns readed packet.
func (c *readHandleCtx) Input() *socket.Packet {
	return c.input
}

// InputHeader returns readed packet header.
func (c *readHandleCtx) InputHeader() *socket.Header {
	return c.input.Header
}

// Output returns writed packet.
func (c *readHandleCtx) Output() *socket.Packet {
	return c.output
}

// Public returns temporary public data of context.
func (c *readHandleCtx) Public() goutil.Map {
	return c.public
}

// PublicLen returns the length of public data of context.
func (c *readHandleCtx) PublicLen() int {
	return c.public.Len()
}

// Uri returns the input packet uri.
func (c *readHandleCtx) Uri() string {
	return c.input.Header.Uri
}

// Path returns the input packet uri path.
func (c *readHandleCtx) Path() string {
	return c.uri.Path
}

// RawQuery returns the input packet uri query string.
func (c *readHandleCtx) RawQuery() string {
	return c.uri.RawQuery
}

// Query returns the input packet uri query object.
func (c *readHandleCtx) Query() url.Values {
	if c.query == nil {
		c.query = c.uri.Query()
	}
	return c.query
}

// SetBodyCodec sets the body codec for response packet.
func (c *readHandleCtx) SetBodyCodec(codecName string) {
	c.output.BodyCodec = codecName
}

// Ip returns the remote addr.
func (c *readHandleCtx) Ip() string {
	return c.sess.RemoteIp()
}

// Be executed synchronously when reading packet
func (c *readHandleCtx) binding(header *socket.Header) (body interface{}) {
	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
			body = nil
		}
	}()
	c.start = time.Now()
	c.pluginContainer = c.sess.peer.pluginContainer
	switch header.Type {
	case TypeReply:
		return c.bindReply(header)

	case TypePush:
		return c.bindPush(header)

	case TypePull:
		return c.bindPull(header)

	default:
		return nil
	}
}

// Be executed asynchronously after readed packet
func (c *readHandleCtx) handle() {
	switch c.input.Header.Type {
	case TypeReply:
		// handles pull reply
		c.handleReply()

	case TypePush:
		//  handles push
		c.handlePush()

	case TypePull:
		// handles and replies pull
		c.handlePull()

	default:
		// if unsupported, disconnected.
		c.output.Header.StatusCode = StatusUnsupportedTx
		c.output.Header.Status = StatusText(StatusUnsupportedTx)
		if c.sess.peer.printBody {
			logformat := "disconnect(%s) due to unsupported type: %d |\nseq: %d |uri: %-30s |\nRECV:\n size: %d\n body[-json]: %s\n"
			Errorf(logformat, c.Ip(), c.input.Header.Type, c.input.Header.Seq, c.input.Header.Uri, c.input.Size, bodyLogBytes(c.input))
		} else {
			logformat := "disconnect(%s) due to unsupported type: %d |\nseq: %d |uri: %-30s |\nRECV:\n size: %d\n"
			Errorf(logformat, c.Ip(), c.input.Header.Type, c.input.Header.Seq, c.input.Header.Uri, c.input.Size)
		}
		go c.sess.Close()
	}
}

func (c *readHandleCtx) bindPush(header *socket.Header) interface{} {
	var err error
	err = c.pluginContainer.PostReadPushHeader(c)
	if err != nil {
		return nil
	}
	c.uri, err = url.Parse(header.Uri)
	if err != nil {
		return nil
	}

	var ok bool
	c.apiType, ok = c.sess.pushRouter.get(c.Path())
	if !ok {
		return nil
	}

	c.pluginContainer = c.apiType.pluginContainer
	c.arg = reflect.New(c.apiType.argElem)
	c.input.Body = c.arg.Interface()

	err = c.pluginContainer.PreReadPushBody(c)
	if err != nil {
		return nil
	}

	return c.input.Body
}

// handlePush handles push.
func (c *readHandleCtx) handlePush() {
	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
		}
		c.cost = time.Since(c.start)
		c.sess.runlog(c.cost, c.input, nil)
	}()

	if c.apiType == nil {
		return
	}

	err := c.pluginContainer.PostReadPushBody(c)
	if err != nil {
		return
	}

	if c.apiType.isUnknown {
		c.apiType.unknownHandleFunc(c)
	} else {
		c.apiType.handleFunc(c, c.arg)
	}
}

func (c *readHandleCtx) bindPull(header *socket.Header) interface{} {
	c.output.Header.Seq = header.Seq
	c.output.Header.Type = TypeReply
	c.output.Header.Uri = header.Uri
	c.output.HeaderCodec = c.input.HeaderCodec
	c.output.Header.Gzip = header.Gzip

	xerr := c.pluginContainer.PostReadPullHeader(c)
	if xerr != nil {
		c.output.Header.StatusCode = xerr.Code()
		c.output.Header.Status = xerr.Text()
		return nil
	}
	var err error
	c.uri, err = url.Parse(header.Uri)
	if err != nil {
		c.output.Header.StatusCode = StatusBadUri
		c.output.Header.Status = err.Error()
		return nil
	}

	var ok bool
	c.apiType, ok = c.sess.pullRouter.get(c.Path())
	if !ok {
		c.output.Header.StatusCode = StatusNotFound
		c.output.Header.Status = StatusText(StatusNotFound)
		return nil
	}

	c.pluginContainer = c.apiType.pluginContainer
	if c.apiType.isUnknown {
		c.input.Body = new([]byte)
	} else {
		c.arg = reflect.New(c.apiType.argElem)
		c.input.Body = c.arg.Interface()
	}

	xerr = c.pluginContainer.PreReadPullBody(c)
	if xerr != nil {
		c.output.Header.StatusCode = xerr.Code()
		c.output.Header.Status = xerr.Text()
		return nil
	}

	c.output.Header.StatusCode = StatusOK
	c.output.Header.Status = StatusText(StatusOK)
	return c.input.Body
}

// handlePull handles and replies pull.
func (c *readHandleCtx) handlePull() {
	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
		}
		c.cost = time.Since(c.start)
		c.sess.runlog(c.cost, c.input, c.output)
	}()

	// handle pull
	if c.output.Header.StatusCode == StatusOK {
		xerr := c.pluginContainer.PostReadPullBody(c)
		if xerr != nil {
			c.output.Header.StatusCode = xerr.Code()
			c.output.Header.Status = xerr.Text()
		} else {
			if c.apiType.isUnknown {
				c.apiType.unknownHandleFunc(c)
			} else {
				c.apiType.handleFunc(c, c.arg)
			}
		}
	}

	// reply pull
	c.pluginContainer.PreWriteReply(c)

	if err := c.sess.write(c.output); err != nil {
		c.output.Header.StatusCode = StatusWriteFailed
		c.output.Header.Status = StatusText(StatusWriteFailed) + ": " + err.Error()
	}

	c.pluginContainer.PostWriteReply(c)
}

func (c *readHandleCtx) bindReply(header *socket.Header) interface{} {
	pullCmd, ok := c.sess.pullCmdMap.Load(header.Seq)
	if !ok {
		Debugf("bindReply() not found: %#v", header)
		return nil
	}
	c.pullCmd = pullCmd.(*PullCmd)
	c.public = c.pullCmd.public

	xerr := c.pluginContainer.PostReadReplyHeader(c)
	if xerr != nil {
		c.pullCmd.xerror = xerr
		return nil
	}
	xerr = c.pluginContainer.PreReadReplyBody(c)
	if xerr != nil {
		c.pullCmd.xerror = xerr
		return nil
	}
	return c.pullCmd.reply
}

// handleReply handles pull reply.
func (c *readHandleCtx) handleReply() {
	if c.pullCmd == nil {
		Debugf("c.pullCmd == nil:\npacket:%#v\nheader%#v", c.input, c.input.Header)
		return
	}
	defer func() {
		if p := recover(); p != nil {
			Errorf("panic:\n%v\n%s", p, goutil.PanicTrace(1))
		}
		c.pullCmd.done()
		c.pullCmd.cost = time.Since(c.pullCmd.start)
		c.sess.runlog(c.pullCmd.cost, c.input, c.pullCmd.output)
	}()
	if c.pullCmd.xerror != nil {
		return
	}
	if c.input.Header.StatusCode != StatusOK {
		c.pullCmd.xerror = NewXerror(c.input.Header.StatusCode, c.input.Header.Status)
	} else if xerr := c.pluginContainer.PostReadReplyBody(c); xerr != nil {
		c.pullCmd.xerror = xerr
	}
}

// InputBodyBytes if the input body binder is []byte type, returns it, else returns nil.
func (c *readHandleCtx) InputBodyBytes() []byte {
	b, ok := c.input.Body.(*[]byte)
	if !ok {
		return nil
	}
	return *b
}

// Bind when the raw body binder is []byte type, now binds the input body to v.
func (c *readHandleCtx) Bind(v interface{}) (string, error) {
	b := c.InputBodyBytes()
	if b == nil {
		return "", nil
	}
	codecName, err := socket.Unmarshal(b, v, c.input.Header.Gzip != gzip.NoCompression)
	if err == nil {
		c.input.Body = v
	}
	return codecName, err
}

// PullCmd the command of the pulling operation's response.
type PullCmd struct {
	sess     *session
	output   *socket.Packet
	reply    interface{}
	xerror   Xerror
	doneChan chan *PullCmd // Strobes when pull is complete.
	start    time.Time
	cost     time.Duration
	public   goutil.Map
}

var _ WriteCtx = new(PullCmd)

// Peer returns the peer.
func (c *PullCmd) Peer() *Peer {
	return c.sess.peer
}

// Session returns the session.
func (c *PullCmd) Session() Session {
	return c.sess
}

// Ip returns the remote addr.
func (c *PullCmd) Ip() string {
	return c.sess.RemoteIp()
}

// Public returns temporary public data of context.
func (c *PullCmd) Public() goutil.Map {
	return c.public
}

// PublicLen returns the length of public data of context.
func (c *PullCmd) PublicLen() int {
	return c.public.Len()
}

// Output returns writed packet.
func (c *PullCmd) Output() *socket.Packet {
	return c.output
}

// Result returns the pull result.
func (c *PullCmd) Result() (interface{}, Xerror) {
	return c.reply, c.xerror
}

// Xerror returns the pull error.
func (c *PullCmd) Xerror() Xerror {
	return c.xerror
}

func (p *PullCmd) cancel() {
	defer func() {
		recover()
	}()
	p.xerror = NewXerror(StatusConnClosed, StatusText(StatusConnClosed))
	p.doneChan <- p
	p.sess.pullCmdMap.Delete(p.output.Header.Seq)
	{
		// free count pull-launch
		p.sess.gracePullCmdWaitGroup.Done()
	}
}

func (p *PullCmd) done() {
	defer func() {
		recover()
	}()
	p.doneChan <- p
	p.sess.pullCmdMap.Delete(p.output.Header.Seq)
	{
		// free count pull-launch
		p.sess.gracePullCmdWaitGroup.Done()
	}
}
