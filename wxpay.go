package wxpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math/rand"
	"sort"
	"strings"
	"time"

	"encoding/hex"
	"errors"
	"github.com/cocotyty/httpclient"
	"net"
	"net/http"
	"reflect"
	"strconv"
)

type BaseRequest struct {
	XMLName  xml.Name `xml:"xml"`
	Sign     string   `xml:"sign"`
	SignType string   `xml:"sign_type"`
	NonceStr string   `xml:"nonce_str"`
}

const (
	HOST               = "https://api.mch.weixin.qq.com"
	SANDBOX            = "/sandbox"
	TransfersPath      = "/mmpaymkttransfers/promotion/transfers"
	TransfersQueryPath = "/mmpaymkttransfers/gettransferinfo"
	RefundPath         = "/secapi/pay/refund"
	AddReceiverPath    = "/pay/profitsharingaddreceiver"
	SharingPath        = "/secapi/pay/profitsharing"
)

type Client struct {
	AppID            string
	MchID            string
	ApiKey           string
	PrivateKeyFile   string
	CertificateFile  string
	PrivateKeyBytes  []byte
	CertificateBytes []byte
	CAFile           string
	CABytes          []byte
	SandBox          bool
	config           *tls.Config
	client           *http.Client
}

func (c *Client) Init() {
	c.config = c.mustGetTlsConfiguration()
	c.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: c.config,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

type ReceiverItem struct {
	Type         string `json:"type"`
	Account      string `json:"account"`
	Name         string `json:"name,omitempty"`
	RelationType string `json:"relation_type,omitempty"`
	Amount       int    `json:"amount,omitempty"`
	Description  string `json:"description,omitempty"`
}

// 分账
type ProfitSharingRequest struct {
	*BaseRequest
	AppID         string `xml:"appid"`
	MchID         string `xml:"mch_id"`
	TransactionId string `xml:"transaction_id"`
	OutOrderNo    string `xml:"out_order_no"`
	Receivers     string `xml:"receivers"`
}

// 分账 添加子账号
type ProfitSharingAddReceiverRequest struct {
	*BaseRequest
	AppID    string `xml:"appid"`
	MchID    string `xml:"mch_id"`
	Receiver string `xml:"receiver"`
}

// 退款 请求
type RefundRequest struct {
	*BaseRequest
	AppID       string `xml:"appid"`
	MchID       string `xml:"mch_id"`
	OutTradeNo  string `xml:"out_trade_no"` //partner_trade_no
	OutRefundNo string `xml:"out_refund_no"`
	TotalFee    int    `xml:"total_fee"`
	RefundFee   int    `xml:"refund_fee"`
	RefundDesc  string `xml:"refund_desc"`
}

type RefundResponse struct {
	XMLName       xml.Name `xml:"xml"`
	ReturnCode    string   `xml:"return_code"`
	ReturnMsg     string   `xml:"return_msg"`
	ResultCode    string   `xml:"result_code"`
	ErrCode       string   `xml:"err_code"`
	ErrCodeDes    string   `xml:"err_code_des"`
	TransactionId string   `xml:"transaction_id"`
	OutTradeNo    string   `xml:"out_trade_no"`
	OutRefundNo   string   `xml:"out_refund_no"`
	RefundId      string   `xml:"refund_id"`
}

// 企业向个人转账的订单查询 请求
type CompanyTransferQueryRequest struct {
	*BaseRequest
	AppID          string `xml:"appid"`
	MchID          string `xml:"mch_id"`
	PartnerTradeNo string `xml:"partner_trade_no"` //partner_trade_no
}

// 企业向个人转账的订单查询 响应
type CompanyTransferQueryResponse struct {
	ReturnCode     string `xml:"return_code"`
	ReturnMsg      string `xml:"return_msg"`
	ResultCode     string `xml:"result_code"`      // 业务结果 	是	SUCCESS	String(16)	SUCCESS/FAIL
	ErrCode        string `xml:"err_code"`         // 错误代码 	否	SYSTEMERROR	String(32)	错误码信息
	ErrCodeDes     string `xml:"err_code_des"`     // 错误代码描述 	否	系统错误	String(128)	结果信息描述
	PartnerTradeNo string `xml:"partner_trade_no"` // 商户单号 	是	10000098201411111234567890	String(28)	商户使用查询API填写的单号的原路返回.
	MchId          string `xml:"mch_id"`           // 商户号 	是	10000098	String(32)	微信支付分配的商户号
	DetailId       string `xml:"detail_id"`        // 付款单号 	是	1000000000201503283103439304	String(32)	调用企业付款API时，微信系统内部产生的单号
	Status         string `xml:"status"`           // 转账状态 	是	SUCCESS	string(16)
	Reason         string `xml:"reason"`           // 失败原因 	否	余额不足	String	如果失败则有失败原因
	Openid         string `xml:"openid"`           // 收款用户openid 	是	oxTWIuGaIt6gTKsQRLau2M0yL16E	 	转账的openid
	TransferName   string `xml:"transfer_name"`    // 收款用户姓名 	否	马华	String	收款用户姓名
	PaymentAmount  string `xml:"payment_amount"`   // 付款金额 	是	5000	int	付款金额单位分）
	TransferTime   string `xml:"transfer_time"`    // 转账时间 	是	2015-04-21 20:00:00	String	发起转账的时间
	Desc           string `xml:"desc"`             // 付款描述 	是	车险理赔	String	付款时候的描述
}
type CompanyTransferRequest struct {
	*BaseRequest

	AppID          string `xml:"mch_appid"`
	MchID          string `xml:"mchid"`
	PartnerTradeNo string `xml:"partner_trade_no"`
	Openid         string `xml:"openid"`
	CheckName      string `xml:"check_name"`
	ReUserName     string `xml:"re_user_name"`
	Amount         string `xml:"amount"`
	Desc           string `xml:"desc"`
	SpbillCreateIp string `xml:"spbill_create_ip"`
}

type CompanyTransferRequestNoCheck struct {
	*BaseRequest

	AppID          string `xml:"mch_appid"`
	MchID          string `xml:"mchid"`
	PartnerTradeNo string `xml:"partner_trade_no"`
	Openid         string `xml:"openid"`
	CheckName      string `xml:"check_name"`
	Amount         string `xml:"amount"`
	Desc           string `xml:"desc"`
	SpbillCreateIp string `xml:"spbill_create_ip"`
}

type CompanyTransferResponse struct {
	XMLName        xml.Name `xml:"xml"`
	ReturnCode     string   `xml:"return_code"`
	ReturnMsg      string   `xml:"return_msg"`
	ResultCode     string   `xml:"result_code"`
	ErrCode        string   `xml:"err_code"`
	ErrCodeDes     string   `xml:"err_code_des"`
	PartnerTradeNo string   `xml:"partner_trade_no"`
	PaymentNo      string   `xml:"payment_no"`
	PaymentTime    string   `xml:"payment_time"`
}

func (c *Client) request() *BaseRequest {
	return &BaseRequest{
		NonceStr: getNonceStr(),
	}
}

func (c *Client) ProfitSharing(req *ProfitSharingRequest) (*RefundResponse, error) {
	req.AppID = c.AppID
	req.MchID = c.MchID
	data, err := c.send(SharingPath, req)
	if err != nil {
		return nil, err
	}
	resp := &RefundResponse{}
	err = xml.Unmarshal(data, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) ProfitSharingAddReceiver(req *ProfitSharingAddReceiverRequest) (*RefundResponse, error) {
	req.AppID = c.AppID
	req.MchID = c.MchID
	data, err := c.sendNoCert(AddReceiverPath, req)
	if err != nil {
		return nil, err
	}
	resp := &RefundResponse{}
	err = xml.Unmarshal(data, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Refund(req *RefundRequest) (*RefundResponse, error) {
	req.AppID = c.AppID
	req.MchID = c.MchID
	data, err := c.send(RefundPath, req)
	if err != nil {
		return nil, err
	}
	resp := &RefundResponse{}
	err = xml.Unmarshal(data, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) CompanyTransfer(req *CompanyTransferRequest) (*CompanyTransferResponse, error) {
	req.AppID = c.AppID
	req.MchID = c.MchID
	data, err := c.send(TransfersPath, req)
	if err != nil {
		return nil, err
	}
	resp := &CompanyTransferResponse{}
	err = xml.Unmarshal(data, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) CompanyTransferNoCheck(req *CompanyTransferRequestNoCheck) (*CompanyTransferResponse, error) {
	req.AppID = c.AppID
	req.MchID = c.MchID
	data, err := c.send(TransfersPath, req)
	if err != nil {
		return nil, err
	}
	resp := &CompanyTransferResponse{}
	err = xml.Unmarshal(data, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) CompanyTransferQuery(req *CompanyTransferQueryRequest) (*CompanyTransferQueryResponse, error) {
	req.AppID = c.AppID
	req.MchID = c.MchID
	data, err := c.send(TransfersQueryPath, req)
	if err != nil {
		return nil, err
	}
	resp := &CompanyTransferQueryResponse{}
	err = xml.Unmarshal(data, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) MAP2XML(m map[string]interface{}) string {
	str := ""
	for k, v := range m {
		switch v.(type) {
		case string:
			str = str + fmt.Sprintf("<%s><![CDATA[%s]]></%s>", k, v, k)
		case int:
			str = str + fmt.Sprintf("<%s><![CDATA[%d]]></%s>", k, v, k)
		case interface{}:
			b, _ := json.Marshal(v)
			str = str + fmt.Sprintf("<%s><![CDATA[%s]]></%s>", k, string(b), k)
		}
	}
	return "<xml>" + str + "</xml>"
}

func (c *Client) format(request interface{}) map[string]interface{} {
	val := reflect.ValueOf(request)
	m := map[string]interface{}{}
	lv := &linkValues{
		keys:   []string{},
		values: map[string]string{},
	}
	err := parseXMLTag2(lv, val)
	if err != nil {
		return m
	}

	sort.Strings(lv.keys)

	for _, v := range lv.keys {
		m[v] = lv.values[v]
	}

	return m
}

// HmacSha256 HMAC-SHA256加密
func (c *Client) HmacSha256(str string, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(str))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}
func (c *Client) sendNoCert(path string, req interface{}) ([]byte, error) {
	url := HOST
	if c.SandBox {
		url += SANDBOX
	}
	url += path

	c.signRequest(req,"HMAC-SHA256")

	data, err := xml.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = bytes.ReplaceAll(data, []byte("&#34;"), []byte(`"`))

	return httpclient.Post(url).Head("Content-Type", "").Body([]byte(data)).Send().Body()
}

func (c *Client) send(path string, req interface{}) ([]byte, error) {
	url := HOST
	if c.SandBox {
		url += SANDBOX
	}
	url += path

	c.signRequest(req,"HMAC-SHA256")

	data, err := xml.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = bytes.ReplaceAll(data, []byte("&#34;"), []byte(`"`))
	return httpclient.New(c.client).Post(url).Body(data).Send().Body()
}
func (c *Client) mustLoadCertificates() (tls.Certificate, *x509.CertPool) {

	var mycert tls.Certificate
	var err error
	if c.PrivateKeyFile != "" {
		privateKeyFile := c.PrivateKeyFile
		certificateFile := c.CertificateFile

		mycert, err = tls.LoadX509KeyPair(certificateFile, privateKeyFile)
		if err != nil {
			panic(err)
		}
	} else {
		mycert, err = tls.X509KeyPair(c.CertificateBytes, c.PrivateKeyBytes)
	}

	if err != nil {
		panic(err)
	}

	var pem []byte
	if c.CAFile != "" {
		caFile := c.CAFile
		pem, err = ioutil.ReadFile(caFile)
		if err != nil {
			panic(err)
		}
	} else {
		pem = c.CABytes
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pem) {
		panic("Failed appending certs")
	}

	return mycert, certPool

}

func (c *Client) mustGetTlsConfiguration() *tls.Config {
	config := &tls.Config{}
	mycert, _ := c.mustLoadCertificates()
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0] = mycert

	//config.RootCAs = certPool
	//config.ClientCAs = certPool
	//
	//config.ClientAuth = tls.RequireAndVerifyClientCert

	//Optional stuff

	//Use only modern ciphers
	config.CipherSuites = []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256}

	//Use only TLS v1.2
	config.MinVersion = tls.VersionTLS12

	//Don't allow session resumption
	config.SessionTicketsDisabled = true
	return config
}

var reqType = reflect.TypeOf(&BaseRequest{})

func parseXMLTag2(lv *linkValues, val reflect.Value) (err error) {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		err = errors.New("must struct")
		return
	}
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Anonymous {
			parseXMLTag2(lv, val.Field(i))
			continue
		}

		name := strings.Split(f.Tag.Get("xml"), ",")[0]

		fieldValue := val.Field(i).Interface()
		var fieldStrValue string
		switch converted := fieldValue.(type) {
		case string:
			fieldStrValue = converted
		case int:
			fieldStrValue = strconv.Itoa(converted)
		case time.Time:
			fieldStrValue = converted.Format("2006-01-02 15:04:05")
		default:
			continue
		}
		lv.keys = append(lv.keys, name)
		lv.values[name] = fieldStrValue
	}
	return nil
}
func parseXMLTag(lv *linkValues, val reflect.Value) (err error) {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		err = errors.New("must struct")
		return
	}
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Anonymous {
			parseXMLTag(lv, val.Field(i))
			continue
		}

		name := strings.Split(f.Tag.Get("xml"), ",")[0]
		if name == "" || name == "sign" {
			continue
		}
		fieldValue := val.Field(i).Interface()
		var fieldStrValue string
		switch converted := fieldValue.(type) {
		case string:
			fieldStrValue = converted
		case int:
			fieldStrValue = strconv.Itoa(converted)
		case time.Time:
			fieldStrValue = converted.Format("2006-01-02 15:04:05")
		default:
			continue
		}
		lv.keys = append(lv.keys, name)
		lv.values[name] = fieldStrValue
	}
	return nil
}

var emptyValue = reflect.Value{}

func (c *Client) injectRequest(val reflect.Value, req *BaseRequest) {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	if field := val.FieldByName("AppID"); field != emptyValue {
		field.SetString(c.AppID)
	}
	if field := val.FieldByName("MchID"); field != emptyValue {
		field.SetString(c.MchID)
	}
	for i := 0; i < val.NumField(); i++ {
		f := typ.Field(i)
		if f.Anonymous && f.Type == reqType {
			val.Field(i).Set(reflect.ValueOf(req))
		}
	}
}

type linkValues struct {
	keys   []string
	values map[string]string
}

func (c *Client) sign(val reflect.Value, signType string) (string, error) {
	lv := &linkValues{
		keys:   []string{},
		values: map[string]string{},
	}
	err := parseXMLTag(lv, val)
	if err != nil {
		return "", err
	}

	sort.Strings(lv.keys)
	bf := bytes.NewBuffer(nil)
	for _, v := range lv.keys {
		bf.WriteString(v)
		bf.WriteByte('=')
		bf.WriteString(lv.values[v])
		bf.WriteByte('&')
	}

	bf.WriteString("key=")
	bf.WriteString(c.ApiKey)
	var res string


	if signType == "HMAC-SHA256" {
		res = c.HmacSha256(bf.String(), c.ApiKey)
	} else {

		bs := md5.Sum(bf.Bytes())
		res = hex.EncodeToString(bs[:])
	}
	return strings.ToUpper(res), nil
}
func (c *Client) signRequest(request interface{}, signType string) (err error) {
	req := c.request()
	req.SignType = signType
	val := reflect.ValueOf(request)
	c.injectRequest(val, req)


	req.Sign, err = c.sign(val, signType)
	return err
}

//获取32位长度的随机数
func getNonceStr() (nonceStr string) {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := 0; i < 32; i++ {
		idx := rand.Intn(len(chars) - 1)
		nonceStr += chars[idx : idx+1]
	}
	return
}

const (
	ReturnCodeSuccess = "SUCCESS"
	ReturnCodeFail    = "FAIL"
)
const (
	ErrCode_NO_AUTH               = "NO_AUTH"                   //没有该接口权限	没有授权请求此api	请关注是否满足接口调用条件
	ErrCode_AMOUNT_LIMIT          = "AMOUNT_LIMIT"              //付款金额不能小于最低限额	付款金额不能小于最低限额	每次付款金额必须大于1元 付款失败，因你已违反《微信支付商户平台使用协议》，单笔单次付款下限已被调整为5元	商户号存在违反协议内容行为，单次付款下限提高	请遵守《微信支付商户平台使用协议》
	ErrCode_PARAM_ERROR           = "PARAM_ERROR"               //参数错误	参数缺失，或参数格式出错，参数不合法等	请查看err_code_des，修改设置错误的参数
	ErrCode_OPENID_ERROR          = "OPENID_ERROR"              //Openid错误	Openid格式错误或者不属于商家公众账号	请核对商户自身公众号appid和用户在此公众号下的openid。
	ErrCode_SEND_FAILED           = "SEND_FAILED"               //付款错误	付款失败，请换单号重试	付款失败，请换单号重试
	ErrCode_NOTENOUGH             = "NOTENOUGH"                 //余额不足	帐号余额不足	请用户充值或更换支付卡后再支付
	ErrCode_SYSTEMERROR           = "SYSTEMERROR"               //系统繁忙，请稍后再试。	系统错误，请重试	请使用原单号以及原请求参数重试，否则可能造成重复支付等资金风险
	ErrCode_NAME_MISMATCH         = "NAME_MISMATCH"             //姓名校验出错	请求参数里填写了需要检验姓名，但是输入了错误的姓名	填写正确的用户姓名
	ErrCode_SIGN_ERROR            = "SIGN_ERROR"                //签名错误	没有按照文档要求进行签名 签名前没有按照要求进行排序。 没有使用商户平台设置的密钥进行签名 参数有空格或者进行了encode后进行签名。
	ErrCode_XML_ERROR             = "XML_ERROR"                 //Post内容出错	Post请求数据不是合法的xml格式内容	修改post的内容
	ErrCode_FATAL_ERROR           = "FATAL_ERROR"               //两次请求参数不一致	两次请求商户单号一样，但是参数不一致	如果想重试前一次的请求，请用原参数重试，如果重新发送，请更换单号。
	ErrCode_FREQ_LIMIT            = "FREQ_LIMIT"                //超过频率限制，请稍后再试。	接口请求频率超时接口限制	请关注接口的使用条件
	ErrCode_MONEY_LIMIT           = "MONEY_LIMIT"               //已经达到今日付款总额上限/已达到付款给此用户额度上限	接口对商户号的每日付款总额，以及付款给同一个用户的总额有限制	请关注接口的付款限额条件
	ErrCode_CA_ERROR              = "CA_ERROR"                  //证书出错	请求没带证书或者带上了错误的证书 到商户平台下载证书 请求的时候带上该证书 V2
	ErrCode_V2_ACCOUNT_SIMPLE_BAN = "V2_ACCOUNT_SIMPLE_BAN	" //无法给非实名用户付款	用户微信支付账户未知名，无法付款	引导用户在微信支付内进行绑卡实名
	ErrCode_PARAM_IS_NOT_UTF8     = "PARAM_IS_NOT_UTF8"         //请求参数中包含非utf8编码字符	接口规范要求所有请求参数都必须为utf8编码	请关注接口使用规范
	ErrCode_SENDNUM_LIMIT         = "SENDNUM_LIMIT"             //该用户今日付款次数超过限制,如有需要请登录微信支付商户平台更改API安全配置
	ErrCode_NOT_FOUND             = "NOT_FOUND"
)
