
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//版权所有（c）2013-2017 BTCSuite开发者
//此源代码的使用由ISC控制
//可以在许可文件中找到的许可证。

package legacyrpc

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcwallet/chain"
	"github.com/btcsuite/btcwallet/wallet"
	"github.com/btcsuite/websocket"
)

type websocketClient struct {
	conn          *websocket.Conn
	authenticated bool
	remoteAddr    string
	allRequests   chan []byte
	responses     chan []byte
quit          chan struct{} //断开时关闭
	wg            sync.WaitGroup
}

func newWebsocketClient(c *websocket.Conn, authenticated bool, remoteAddr string) *websocketClient {
	return &websocketClient{
		conn:          c,
		authenticated: authenticated,
		remoteAddr:    remoteAddr,
		allRequests:   make(chan []byte),
		responses:     make(chan []byte),
		quit:          make(chan struct{}),
	}
}

func (c *websocketClient) send(b []byte) error {
	select {
	case c.responses <- b:
		return nil
	case <-c.quit:
		return errors.New("websocket client disconnected")
	}
}

//服务器保存RPC服务器可能需要访问的项目（auth，
//配置、关闭等）
type Server struct {
	httpServer    http.Server
	wallet        *wallet.Wallet
	walletLoader  *wallet.Loader
	chainClient   chain.Interface
	handlerLookup func(string) (requestHandler, bool)
	handlerMu     sync.Mutex

	listeners []net.Listener
	authsha   [sha256.Size]byte
	upgrader  websocket.Upgrader

maxPostClients      int64 //最大并发HTTP Post客户端数。
maxWebsocketClients int64 //最大并发WebSocket客户端数。

	wg      sync.WaitGroup
	quit    chan struct{}
	quitMtx sync.Mutex

	requestShutdownChan chan struct{}
}

//如果HTTP认证被拒绝，jsonAuthfail将向客户机发送一条消息。
func jsonAuthFail(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Basic realm="btcwallet RPC"`)
	http.Error(w, "401 Unauthorized.", http.StatusUnauthorized)
}

//new server为旧的RPC客户端连接创建新的服务器，
//HTTP Post和WebSocket。
func NewServer(opts *Options, walletLoader *wallet.Loader, listeners []net.Listener) *Server {
	serveMux := http.NewServeMux()
	const rpcAuthTimeoutSeconds = 10

	server := &Server{
		httpServer: http.Server{
			Handler: serveMux,

//无法完成初始连接的超时连接
//在允许的时间范围内握手。
			ReadTimeout: time.Second * rpcAuthTimeoutSeconds,
		},
		walletLoader:        walletLoader,
		maxPostClients:      opts.MaxPOSTClients,
		maxWebsocketClients: opts.MaxWebsocketClients,
		listeners:           listeners,
//HTTP基本身份验证字符串的哈希用于常量
//时间比较。
		authsha: sha256.Sum256(httpBasicAuth(opts.Username, opts.Password)),
		upgrader: websocket.Upgrader{
//允许所有来源。
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		quit:                make(chan struct{}),
		requestShutdownChan: make(chan struct{}, 1),
	}

	serveMux.Handle("/", throttledFn(opts.MaxPOSTClients,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Connection", "close")
			w.Header().Set("Content-Type", "application/json")
			r.Close = true

			if err := server.checkAuthHeader(r); err != nil {
				log.Warnf("Unauthorized client connection attempt")
				jsonAuthFail(w)
				return
			}
			server.wg.Add(1)
			server.postClientRPC(w, r)
			server.wg.Done()
		}))

	serveMux.Handle("/ws", throttledFn(opts.MaxWebsocketClients,
		func(w http.ResponseWriter, r *http.Request) {
			authenticated := false
			switch server.checkAuthHeader(r) {
			case nil:
				authenticated = true
			case ErrNoAuth:
//没有什么
			default:
//如果提供了auth但不正确，而不是简单地
//如果丢失，请立即终止连接。
				log.Warnf("Disconnecting improperly authorized " +
					"websocket client")
				jsonAuthFail(w)
				return
			}

			conn, err := server.upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Warnf("Cannot websocket upgrade client %s: %v",
					r.RemoteAddr, err)
				return
			}
			wsc := newWebsocketClient(conn, authenticated, r.RemoteAddr)
			server.websocketClientRPC(wsc)
		}))

	for _, lis := range listeners {
		server.serve(lis)
	}

	return server
}

//httpbasicauth返回HTTP基本身份验证的utf-8字节
//字符串：
//
//“基本”+base64（用户名+“：”+密码）
func httpBasicAuth(username, password string) []byte {
	const header = "Basic "
	base64 := base64.StdEncoding

	b64InputLen := len(username) + len(":") + len(password)
	b64Input := make([]byte, 0, b64InputLen)
	b64Input = append(b64Input, username...)
	b64Input = append(b64Input, ':')
	b64Input = append(b64Input, password...)

	output := make([]byte, len(header)+base64.EncodedLen(b64InputLen))
	copy(output, header)
	base64.Encode(output[len(header):], b64Input)
	return output
}

//serve为旧版JSON-RPC服务器提供HTTP POST和WebSocket RPC。
//此函数不会在lis.accept上阻塞。
func (s *Server) serve(lis net.Listener) {
	s.wg.Add(1)
	go func() {
		log.Infof("Listening on %s", lis.Addr())
		err := s.httpServer.Serve(lis)
		log.Tracef("Finished serving RPC: %v", err)
		s.wg.Done()
	}()
}

//RegisterWallet将旧版RPC服务器与Wallet关联。这个
//必须先调用函数，客户端才能调用任何钱包RPC。
func (s *Server) RegisterWallet(w *wallet.Wallet) {
	s.handlerMu.Lock()
	s.wallet = w
	s.handlerMu.Unlock()
}

//停止通过停止并断开所有连接而优雅地关闭RPC服务器
//客户端，断开链服务器连接，关闭钱包
//帐户文件。在关闭完成之前，这将一直阻塞。
func (s *Server) Stop() {
	s.quitMtx.Lock()
	select {
	case <-s.quit:
		s.quitMtx.Unlock()
		return
	default:
	}

//停止连接的钱包和链服务器（如果有）。
	s.handlerMu.Lock()
	wallet := s.wallet
	chainClient := s.chainClient
	s.handlerMu.Unlock()
	if wallet != nil {
		wallet.Stop()
	}
	if chainClient != nil {
		chainClient.Stop()
	}

//停止所有听众。
	for _, listener := range s.listeners {
		err := listener.Close()
		if err != nil {
			log.Errorf("Cannot close listener `%s`: %v",
				listener.Addr(), err)
		}
	}

//向其余Goroutines发出停止信号。
	close(s.quit)
	s.quitMtx.Unlock()

//首先等待钱包和链服务器停止，如果他们
//曾经被设定。
	if wallet != nil {
		wallet.WaitForShutdown()
	}
	if chainClient != nil {
		chainClient.WaitForShutdown()
	}

//等待所有剩余的goroutine退出。
	s.wg.Wait()
}

//setchainserver设置完全运行
//功能性比特币钱包RPC服务器。可以调用此函数来启用RPC
//即使在设置已加载的钱包之前也要通过，但钱包的RPC客户端
//优先考虑。
func (s *Server) SetChainServer(chainClient chain.Interface) {
	s.handlerMu.Lock()
	s.chainClient = chainClient
	s.handlerMu.Unlock()
}

//handlerClosing为处理给定的
//方法。这可能是由btcwallet直接处理的请求，或者
//通过将请求向下传递到BTCD来处理的链服务器请求。
//
//注意：这些处理程序不处理特殊情况，例如身份验证
//方法。每个方法都必须事先检查（方法已经
//已知）并进行相应处理。
func (s *Server) handlerClosure(request *btcjson.Request) lazyHandler {
	s.handlerMu.Lock()
//在锁保持不变的情况下，为闭包复制这些指针。
	wallet := s.wallet
	chainClient := s.chainClient
	if wallet != nil && chainClient == nil {
		chainClient = wallet.ChainClient()
		s.chainClient = chainClient
	}
	s.handlerMu.Unlock()

	return lazyApplyHandler(request, wallet, chainClient)
}

//errnoauth表示身份验证无法成功的错误
//由于缺少授权HTTP头。
var ErrNoAuth = errors.New("no auth")

//checkAuthHeader检查客户端提供的HTTP基本身份验证
//在HTTP请求R中。如果请求不正确，则错误为errnoauth。
//包含授权头，或者如果
//提供了身份验证，但不正确。
//
//这种检查是时间常数。
func (s *Server) checkAuthHeader(r *http.Request) error {
	authhdr := r.Header["Authorization"]
	if len(authhdr) == 0 {
		return ErrNoAuth
	}

	authsha := sha256.Sum256([]byte(authhdr[0]))
	cmp := subtle.ConstantTimeCompare(authsha[:], s.authsha[:])
	if cmp != 1 {
		return errors.New("bad auth")
	}
	return nil
}

//throttledfn包装一个http.handlerFunc，并限制并发活动
//当超过阈值时，通过HTTP 429响应客户机。
func throttledFn(threshold int64, f http.HandlerFunc) http.Handler {
	return throttled(threshold, f)
}

//Throttled使用限制并发活动的HTTP.Handler进行包装
//当超过阈值时，通过HTTP 429响应客户机。
func throttled(threshold int64, h http.Handler) http.Handler {
	var active int64

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt64(&active, 1)
		defer atomic.AddInt64(&active, -1)

		if current-1 >= threshold {
			log.Warnf("Reached threshold of %d concurrent active clients", threshold)
			http.Error(w, "429 Too Many Requests", 429)
			return
		}

		h.ServeHTTP(w, r)
	})
}

//SanitizeRequest返回请求的已清理字符串，该字符串可能是
//安全记录。它旨在删除私钥、密码和任何
//在将请求参数保存到日志文件之前，请求参数中的其他机密。
func sanitizeRequest(r *btcjson.Request) string {
//这些被认为是不安全的日志，所以清理参数。
	switch r.Method {
	case "encryptwallet", "importprivkey", "importwallet",
		"signrawtransaction", "walletpassphrase",
		"walletpassphrasechange":

		return fmt.Sprintf(`{"id":%v,"method":"%s","params":SANITIZED %d parameters}`,
			r.ID, r.Method, len(r.Params))
	}

	return fmt.Sprintf(`{"id":%v,"method":"%s","params":%v}`, r.ID,
		r.Method, r.Params)
}

//id pointer返回指向传递的ID的指针，如果接口为nil，则返回nil。
//接口指针通常是错误操作的红色标志，
//但这只是为了解决BTCJSON的一个奇怪问题，
//它使用空的接口指针作为响应ID。
func idPointer(id interface{}) (p *interface{}) {
	if id != nil {
		p = &id
	}
	return
}

//invalidauth检查websocket请求是否有效（parsable）
//验证请求并检查提供的用户名和密码短语
//针对服务器身份验证。
func (s *Server) invalidAuth(req *btcjson.Request) bool {
	cmd, err := btcjson.UnmarshalCmd(req)
	if err != nil {
		return false
	}
	authCmd, ok := cmd.(*btcjson.AuthenticateCmd)
	if !ok {
		return false
	}
//检查凭证。
	login := authCmd.Username + ":" + authCmd.Passphrase
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
	authSha := sha256.Sum256([]byte(auth))
	return subtle.ConstantTimeCompare(authSha[:], s.authsha[:]) != 1
}

func (s *Server) websocketClientRead(wsc *websocketClient) {
	for {
		_, request, err := wsc.conn.ReadMessage()
		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				log.Warnf("Websocket receive failed from client %s: %v",
					wsc.remoteAddr, err)
			}
			close(wsc.allRequests)
			break
		}
		wsc.allRequests <- request
	}
}

func (s *Server) websocketClientRespond(wsc *websocketClient) {
//使用读取退出通道的for select而不是
//为范围提供干净的关闭。这是必要的，因为
//websocketclientread（发送到allrequests chan）未关闭
//如果远程WebSocket客户端仍然处于
//有联系的。
out:
	for {
		select {
		case reqBytes, ok := <-wsc.allRequests:
			if !ok {
//客户端已断开连接
				break out
			}

			var req btcjson.Request
			err := json.Unmarshal(reqBytes, &req)
			if err != nil {
				if !wsc.authenticated {
//立即断开。
					break out
				}
				resp := makeResponse(req.ID, nil,
					btcjson.ErrRPCInvalidRequest)
				mresp, err := json.Marshal(resp)
//我们希望元帅能成功。如果它
//不，它表示一些不可封送的
//键入响应。
				if err != nil {
					panic(err)
				}
				err = wsc.send(mresp)
				if err != nil {
					break out
				}
				continue
			}

			if req.Method == "authenticate" {
				if wsc.authenticated || s.invalidAuth(&req) {
//立即断开。
					break out
				}
				wsc.authenticated = true
				resp := makeResponse(req.ID, nil, nil)
//希望永远不会失败。
				mresp, err := json.Marshal(resp)
				if err != nil {
					panic(err)
				}
				err = wsc.send(mresp)
				if err != nil {
					break out
				}
				continue
			}

			if !wsc.authenticated {
//立即断开。
				break out
			}

			switch req.Method {
			case "stop":
				resp := makeResponse(req.ID,
					"btcwallet stopping.", nil)
				mresp, err := json.Marshal(resp)
//希望永远不会失败。
				if err != nil {
					panic(err)
				}
				err = wsc.send(mresp)
				if err != nil {
					break out
				}
				s.requestProcessShutdown()
				break

			default:
req := req //关闭副本
				f := s.handlerClosure(&req)
				wsc.wg.Add(1)
				go func() {
					resp, jsonErr := f()
					mresp, err := btcjson.MarshalResponse(req.ID, resp, jsonErr)
					if err != nil {
						log.Errorf("Unable to marshal response: %v", err)
					} else {
						_ = wsc.send(mresp)
					}
					wsc.wg.Done()
				}()
			}

		case <-s.quit:
			break out
		}
	}

//完成所有处理程序goroutine后允许客户端断开连接
	wsc.wg.Wait()
	close(wsc.responses)
	s.wg.Done()
}

func (s *Server) websocketClientSend(wsc *websocketClient) {
	const deadline time.Duration = 2 * time.Second
out:
	for {
		select {
		case response, ok := <-wsc.responses:
			if !ok {
//客户端已断开连接
				break out
			}
			err := wsc.conn.SetWriteDeadline(time.Now().Add(deadline))
			if err != nil {
				log.Warnf("Cannot set write deadline on "+
					"client %s: %v", wsc.remoteAddr, err)
			}
			err = wsc.conn.WriteMessage(websocket.TextMessage,
				response)
			if err != nil {
				log.Warnf("Failed websocket send to client "+
					"%s: %v", wsc.remoteAddr, err)
				break out
			}

		case <-s.quit:
			break out
		}
	}
	close(wsc.quit)
	log.Infof("Disconnected websocket client %s", wsc.remoteAddr)
	s.wg.Done()
}

//websocketclientrpc启动goroutine，通过
//单个客户端的WebSocket连接。
func (s *Server) websocketClientRPC(wsc *websocketClient) {
	log.Infof("New websocket client %s", wsc.remoteAddr)

//清除WebSocket被劫持之前设置的读取截止时间
//连接。
	if err := wsc.conn.SetReadDeadline(time.Time{}); err != nil {
		log.Warnf("Cannot remove read deadline: %v", err)
	}

//WebSocketClientRead故意不与WaitGroup一起运行
//所以在关机时会被忽略。这是为了防止在
//当Goroutine在读取
//如果客户端仍然连接，则为WebSocket连接。
	go s.websocketClientRead(wsc)

	s.wg.Add(2)
	go s.websocketClientRespond(wsc)
	go s.websocketClientSend(wsc)

	<-wsc.quit
}

//maxrequestsize指定请求正文中的最大字节数
//可以从客户机读取。目前限制为4MB。
const maxRequestSize = 1024 * 1024 * 4

//PostclientRpc处理并回复JSON-RPC客户机请求。
func (s *Server) postClientRPC(w http.ResponseWriter, r *http.Request) {
	body := http.MaxBytesReader(w, r.Body, maxRequestSize)
	rpcRequest, err := ioutil.ReadAll(body)
	if err != nil {
//TODO:如果底层阅读器出错怎么办？
		http.Error(w, "413 Request Too Large.",
			http.StatusRequestEntityTooLarge)
		return
	}

//首先检查Wallet是否有此请求方法的处理程序。
//如果未找到，则请求将发送到链服务器以进一步
//处理。在检查方法时，不允许进行身份验证
//请求，因为它们对HTTP Post客户端无效。
	var req btcjson.Request
	err = json.Unmarshal(rpcRequest, &req)
	if err != nil {
		resp, err := btcjson.MarshalResponse(req.ID, nil, btcjson.ErrRPCInvalidRequest)
		if err != nil {
			log.Errorf("Unable to marshal response: %v", err)
			http.Error(w, "500 Internal Server Error",
				http.StatusInternalServerError)
			return
		}
		_, err = w.Write(resp)
		if err != nil {
			log.Warnf("Cannot write invalid request request to "+
				"client: %v", err)
		}
		return
	}

//从请求创建响应和错误。两个特例
//对身份验证和停止请求方法进行处理。
	var res interface{}
	var jsonErr *btcjson.RPCError
	var stop bool
	switch req.Method {
	case "authenticate":
//放弃它。
		return
	case "stop":
		stop = true
		res = "btcwallet stopping"
	default:
		res, jsonErr = s.handlerClosure(&req)()
	}

//封送。
	mresp, err := btcjson.MarshalResponse(req.ID, res, jsonErr)
	if err != nil {
		log.Errorf("Unable to marshal response: %v", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	_, err = w.Write(mresp)
	if err != nil {
		log.Warnf("Unable to respond to client: %v", err)
	}

	if stop {
		s.requestProcessShutdown()
	}
}

func (s *Server) requestProcessShutdown() {
	select {
	case s.requestShutdownChan <- struct{}{}:
	default:
	}
}

//RequestProcessShutdown返回一个通道，当
//客户端请求远程关闭。
func (s *Server) RequestProcessShutdown() <-chan struct{} {
	return s.requestShutdownChan
}
