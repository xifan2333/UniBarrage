package xiaohongshu

import (
	"crypto/md5"
	"crypto/rc4"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math/rand"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	xsecAppID      = "xhs-pc-web"
	xysPrefix      = "XYS_"
	x3Prefix       = "mns0301_"
	b1SecretKey    = "xhswebmplfbt"
	publicUA       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36 Edg/142.0.0.0"
	customB64Alpha = "ZmserbBoHQtNP+wOcza/LpngG8yJq42KWYj0DSfdikx3VT16IlUAFM97hECvuRX5"
	x3B64Alpha     = "MfgqrsbcyzPQRStuvC7mn501HIJBo2DEFTKdeNOwxWXYZap89+/A4UVLhijkl63G"
	stdB64Alpha    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	hexKey         = "71a302257793271ddd273bcee3e4b98d9d7935e1da33f5765e2ea8afb6dc77a51a499d23b67c20660025860cbf13d4540d92497f58686c574e508f46e1956344f39139bf4faf22a3eef120b79258145b2feb5193b6478669961298e79bedca646e1a693a926154a5a7a1bd1cf0dedb742f917a747a1e388b234f2277516db7116035439730fa61e9822a0eca7bff72d8"
)

var (
	versionBytes = []byte{121, 104, 96, 41}
	a3Prefix     = []byte{2, 97, 51, 16}
	envTable     = []int{115, 248, 83, 102, 103, 201, 181, 131, 99, 94, 4, 68, 250, 132, 21}
	envChecks    = []int{0, 1, 18, 1, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0}
	hashIV       = [4]uint32{1831565813, 461845907, 2246822507, 3266489909}
	hexKeyBytes  []byte
	customEncode *strings.Replacer
	x3Encode     *strings.Replacer
	polyTable    [256]uint32
)

func init() {
	var err error
	hexKeyBytes, err = hex.DecodeString(hexKey)
	if err != nil {
		panic(err)
	}
	customEncode = buildReplacer(stdB64Alpha, customB64Alpha)
	x3Encode = buildReplacer(stdB64Alpha, x3B64Alpha)
	// JS-style CRC table
	const poly = 0xEDB88320
	for d := 0; d < 256; d++ {
		r := uint32(d)
		for i := 0; i < 8; i++ {
			if r&1 != 0 {
				r = (r >> 1) ^ poly
			} else {
				r >>= 1
			}
		}
		polyTable[d] = r
	}
}

func buildReplacer(from, to string) *strings.Replacer {
	args := make([]string, 0, len(from)*2)
	for i := 0; i < len(from); i++ {
		args = append(args, string(from[i]), string(to[i]))
	}
	return strings.NewReplacer(args...)
}

func encodeCustomB64(data []byte) string {
	return customEncode.Replace(base64.StdEncoding.EncodeToString(data))
}

func encodeX3B64(data []byte) string {
	return x3Encode.Replace(base64.StdEncoding.EncodeToString(data))
}

// GenerateA1 builds a 52-char a1 cookie (xhshow algorithm).
func GenerateA1() string {
	tsHex := strconv.FormatInt(time.Now().UnixMilli(), 16)
	const charset = "abcdefghijklmnopqrstuvwxyz1234567890"
	var b strings.Builder
	b.WriteString(tsHex)
	for i := 0; i < 30; i++ {
		b.WriteByte(charset[rand.Intn(len(charset))])
	}
	b.WriteString("50")
	b.WriteString("000")
	aPart := b.String()
	crc := crc32.ChecksumIEEE([]byte(aPart))
	out := aPart + strconv.FormatUint(uint64(crc), 10)
	if len(out) > 52 {
		out = out[:52]
	}
	return out
}

// GenerateWebID = md5(a1)
func GenerateWebID(a1 string) string {
	sum := md5.Sum([]byte(a1))
	return hex.EncodeToString(sum[:])
}

func buildContentString(method, uri string, payload any) (string, error) {
	if strings.EqualFold(method, "POST") {
		if payload == nil {
			return uri + "{}", nil
		}
		// Prefer exact bytes when caller already marshaled (json.RawMessage / []byte)
		switch t := payload.(type) {
		case json.RawMessage:
			return uri + string(t), nil
		case []byte:
			return uri + string(t), nil
		case string:
			return uri + t, nil
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		return uri + string(b), nil
	}
	// GET
	if payload == nil {
		return uri, nil
	}
	m, ok := payload.(map[string]any)
	if !ok {
		return uri, nil
	}
	if len(m) == 0 {
		return uri, nil
	}
	// Stable alphabetical key order — must match url.Values.Encode() on the wire.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(m))
	for _, k := range keys {
		v := m[k]
		var vs string
		switch t := v.(type) {
		case nil:
			vs = ""
		case []any:
			ss := make([]string, len(t))
			for i, x := range t {
				ss[i] = fmt.Sprint(x)
			}
			vs = strings.Join(ss, ",")
		default:
			vs = fmt.Sprint(t)
		}
		// xhshow GET: urllib.parse.quote(value, safe=",")
		parts = append(parts, k+"="+url.QueryEscape(vs))
	}
	return uri + "?" + strings.Join(parts, "&"), nil
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func intToLE(val uint64, length int) []byte {
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		out[i] = byte(val & 0xFF)
		val >>= 8
	}
	return out
}

func rotl32(val uint32, n uint) uint32 {
	return (val << n) | (val >> (32 - n))
}

func customHashV2(input []byte) []byte {
	s0, s1, s2, s3 := hashIV[0], hashIV[1], hashIV[2], hashIV[3]
	length := len(input)
	s0 ^= uint32(length)
	s1 ^= uint32(length) << 8
	s2 ^= uint32(length) << 16
	s3 ^= uint32(length) << 24
	for i := 0; i < length/8; i++ {
		v0 := binary.LittleEndian.Uint32(input[i*8 : i*8+4])
		v1 := binary.LittleEndian.Uint32(input[i*8+4 : i*8+8])
		s0 = rotl32((s0+v0)^s2, 7)
		s1 = rotl32((v0^s1)+s3, 11)
		s2 = rotl32((s2+v1)^s0, 13)
		s3 = rotl32((s3^v1)+s1, 17)
	}
	t0 := s0 ^ uint32(length)
	t1 := s1 ^ t0
	t2 := s2 + t1
	t3 := s3 ^ t2
	rotT0 := rotl32(t0, 9)
	rotT1 := rotl32(t1, 13)
	rotT2 := rotl32(t2, 17)
	rotT3 := rotl32(t3, 19)
	s0 = rotT0 + rotT2
	s1 = rotT1 ^ rotT3
	s2 = rotT2 + s0
	s3 = rotT3 ^ s1
	out := make([]byte, 16)
	binary.LittleEndian.PutUint32(out[0:], s0)
	binary.LittleEndian.PutUint32(out[4:], s1)
	binary.LittleEndian.PutUint32(out[8:], s2)
	binary.LittleEndian.PutUint32(out[12:], s3)
	return out
}

func buildPayloadArray(hexParam, hexMD5Path, a1, appID, stringParam string, ts float64) []byte {
	seed := rand.Uint32()
	seedByte := byte(seed & 0xFF)
	payload := make([]byte, 0, 144)
	payload = append(payload, versionBytes...)
	payload = append(payload, intToLE(uint64(seed), 4)...)
	tsMs := uint64(ts * 1000)
	tsBytes := intToLE(tsMs, 8)
	payload = append(payload, tsBytes...)

	timeOffset := 10 + rand.Intn(41) // 10..50
	effTs := uint64((ts - float64(timeOffset)) * 1000)
	payload = append(payload, intToLE(effTs, 8)...)
	seq := 15 + rand.Intn(36) // 15..50
	payload = append(payload, intToLE(uint64(seq), 4)...)
	win := 1000 + rand.Intn(201) // 1000..1200
	payload = append(payload, intToLE(uint64(win), 4)...)
	uriLen := len([]byte(stringParam))
	payload = append(payload, intToLE(uint64(uriLen), 4)...)

	md5Bytes, _ := hex.DecodeString(hexParam)
	for i := 0; i < 8 && i < len(md5Bytes); i++ {
		payload = append(payload, md5Bytes[i]^seedByte)
	}

	a1b := []byte(a1)
	if len(a1b) > 52 {
		a1b = a1b[:52]
	}
	if len(a1b) < 52 {
		pad := make([]byte, 52-len(a1b))
		a1b = append(a1b, pad...)
	}
	payload = append(payload, byte(len(a1b)))
	payload = append(payload, a1b...)

	appb := []byte(appID)
	if len(appb) > 10 {
		appb = appb[:10]
	}
	if len(appb) < 10 {
		pad := make([]byte, 10-len(appb))
		appb = append(appb, pad...)
	}
	payload = append(payload, byte(len(appb)))
	payload = append(payload, appb...)

	// part11
	payload = append(payload, 1, seedByte^byte(envTable[0]))
	for i := 1; i < 15; i++ {
		payload = append(payload, byte(envTable[i]^envChecks[i]))
	}

	md5Path, _ := hex.DecodeString(hexMD5Path)
	hashIn := append(append([]byte{}, tsBytes...), md5Path...)
	h := customHashV2(hashIn)
	payload = append(payload, a3Prefix...)
	for _, b := range h {
		payload = append(payload, b^seedByte)
	}
	if len(payload) > 144 {
		payload = payload[:144]
	}
	return payload
}

func xorTransform(src []byte) []byte {
	out := make([]byte, len(src))
	for i := range src {
		if i < len(hexKeyBytes) {
			out[i] = src[i] ^ hexKeyBytes[i]
		} else {
			out[i] = src[i]
		}
	}
	return out
}

// SignXS generates x-s (XYS_…) for edith APIs.
func SignXS(method, uri string, a1 string, payload any, ts float64) (string, error) {
	if ts == 0 {
		ts = float64(time.Now().UnixMilli()) / 1000.0
	}
	content, err := buildContentString(method, uri, payload)
	if err != nil {
		return "", err
	}
	d := md5Hex(content)
	m := d
	if strings.EqualFold(method, "POST") {
		m = md5Hex(uri)
	}
	arr := buildPayloadArray(d, m, a1, xsecAppID, content, ts)
	xored := xorTransform(arr)
	if len(xored) > 144 {
		xored = xored[:144]
	}
	x3 := encodeX3B64(xored)
	sigData := map[string]string{
		"x0": "4.3.5",
		"x1": xsecAppID,
		"x2": "Windows",
		"x3": x3Prefix + x3,
		"x4": "object",
	}
	raw, _ := json.Marshal(sigData)
	return xysPrefix + encodeCustomB64(raw), nil
}

// crc32JS matches xhshow CRC32.crc32_js_int (signed).
func crc32JS(data string) int32 {
	c := uint32(0xFFFFFFFF)
	for i := 0; i < len(data); i++ {
		b := data[i]
		c = polyTable[(c&0xFF)^uint32(b)] ^ (c >> 8)
	}
	u := ((0xFFFFFFFF ^ c) ^ 0xEDB88320) & 0xFFFFFFFF
	return int32(u)
}

func generateB1(a1 string, cookies map[string]string) string {
	// Minimal stable fingerprint subset used by b1
	fp := map[string]any{
		"x33": "0",
		"x34": "0",
		"x35": "0",
		"x36": fmt.Sprintf("%d", 1+rand.Intn(20)),
		"x37": "0|0|0|0|0|0|0|0|0|1|0|0|0|0|0|0|0|0|1|0|0|0|0|0",
		"x38": "0|0|1|0|1|0|0|0|0|0|1|0|1|0|1|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0",
		"x39": 0,
		"x42": "3.4.4",
		"x43": md5Hex(fmt.Sprintf("%d", rand.Int63())),
		"x44": fmt.Sprintf("%d", time.Now().UnixMilli()),
		"x45": "__SEC_CAV__1-1-1-1-1|__SEC_WSA__|",
		"x46": "false",
		"x48": "",
		"x49": "{list:[],type:}",
		"x50": "",
		"x51": "",
		"x52": "",
		"x82": "_0x17a2|_0x1954",
	}
	_ = a1
	_ = cookies
	b1JSON, _ := json.Marshal(fp)
	cipher, err := rc4.NewCipher([]byte(b1SecretKey))
	if err != nil {
		return ""
	}
	ct := make([]byte, len(b1JSON))
	cipher.XORKeyStream(ct, b1JSON)
	// latin1 string then quote
	latin1 := string(ct)
	encoded := url.QueryEscape(latin1)
	// xhshow safe="!*'()~_-" — QueryEscape is stricter; undo over-encoding of those
	for _, ch := range []string{"!", "*", "'", "(", ")", "~", "_", "-"} {
		encoded = strings.ReplaceAll(encoded, url.QueryEscape(ch), ch)
	}
	// parse %XX and trailing chars into bytes
	var b []byte
	parts := strings.Split(encoded, "%")
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	} else if len(parts) > 0 {
		// leading unescaped
		for _, c := range parts[0] {
			b = append(b, byte(c))
		}
		parts = parts[1:]
	}
	for _, p := range parts {
		if len(p) < 2 {
			continue
		}
		v, err := strconv.ParseUint(p[:2], 16, 8)
		if err != nil {
			continue
		}
		b = append(b, byte(v))
		for _, c := range p[2:] {
			b = append(b, byte(c))
		}
	}
	return encodeCustomB64(b)
}

// SignXSCommon generates x-s-common.
func SignXSCommon(cookies map[string]string) string {
	a1 := cookies["a1"]
	b1 := generateB1(a1, cookies)
	x9 := crc32JS(b1)
	signStruct := map[string]any{
		"s0":  5,
		"s1":  "",
		"x0":  "1",
		"x1":  "4.3.5",
		"x2":  "Windows",
		"x3":  xsecAppID,
		"x4":  "4.86.0",
		"x5":  a1,
		"x6":  "",
		"x7":  "",
		"x8":  b1,
		"x9":  x9,
		"x10": 0,
		"x11": "normal",
	}
	raw, _ := json.Marshal(signStruct)
	return encodeCustomB64(raw)
}

func randomHex(n int) string {
	const hexchars = "abcdef0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = hexchars[rand.Intn(len(hexchars))]
	}
	return string(b)
}

func b3TraceID() string { return randomHex(16) }

func xrayTraceID(tsMs int64) string {
	seq := rand.Intn(1 << 23)
	part1 := fmt.Sprintf("%016x", (uint64(tsMs)<<23)|uint64(seq))
	return part1 + randomHex(16)
}

// SignHeaders returns edith signature headers for method/uri.
func SignHeaders(method, uri string, cookies map[string]string, payload any) (map[string]string, error) {
	a1 := cookies["a1"]
	if a1 == "" {
		return nil, fmt.Errorf("missing a1 cookie")
	}
	ts := float64(time.Now().UnixMilli()) / 1000.0
	xs, err := SignXS(method, uri, a1, payload, ts)
	if err != nil {
		return nil, err
	}
	tsMs := int64(ts * 1000)
	return map[string]string{
		"x-s":            xs,
		"x-s-common":     SignXSCommon(cookies),
		"x-t":            strconv.FormatInt(tsMs, 10),
		"x-b3-traceid":   b3TraceID(),
		"x-xray-traceid": xrayTraceID(tsMs),
		"x-mns":          "unload",
		"xy-direction":   strconv.Itoa(10 + rand.Intn(91)),
	}, nil
}
