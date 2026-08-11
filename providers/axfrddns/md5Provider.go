package axfrddns

import (
	"crypto/hmac"
	"crypto/md5" //#nosec
	"encoding/base64"
	"encoding/hex"

	dnsv1 "github.com/miekg/dns"
)

type md5Provider string

func fromBase64(s []byte) (buf []byte, err error) {
	buflen := base64.StdEncoding.DecodedLen(len(s))
	buf = make([]byte, buflen)
	n, err := base64.StdEncoding.Decode(buf, s)
	buf = buf[:n]
	return
}

func (key md5Provider) Generate(msg []byte, _ *dnsv1.TSIG) ([]byte, error) {
	rawsecret, err := fromBase64([]byte(key))
	if err != nil {
		return nil, err
	}
	h := hmac.New(md5.New, rawsecret)

	h.Write(msg)
	return h.Sum(nil), nil
}

func (key md5Provider) Verify(msg []byte, t *dnsv1.TSIG) error {
	b, err := key.Generate(msg, t)
	if err != nil {
		return err
	}
	mac, err := hex.DecodeString(t.MAC)
	if err != nil {
		return err
	}
	if !hmac.Equal(b, mac) {
		return dnsv1.ErrSig
	}
	return nil
}
