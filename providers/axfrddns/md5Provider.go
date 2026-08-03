package axfrddns

import (
	"crypto/hmac"
	"crypto/md5" //#nosec
	"encoding/hex"

	dnsv2 "codeberg.org/miekg/dns"
)

// md5Provider signs and verifies TSIGs with HMAC-MD5.
type md5Provider []byte

func (key md5Provider) Key() []byte { return key }

func (key md5Provider) Sign(_ *dnsv2.TSIG, msg []byte, _ dnsv2.TSIGOption) ([]byte, error) {
	h := hmac.New(md5.New, key)
	h.Write(msg)
	return h.Sum(nil), nil
}

func (key md5Provider) Verify(t *dnsv2.TSIG, msg []byte, options dnsv2.TSIGOption) error {
	b, err := key.Sign(t, msg, options)
	if err != nil {
		return err
	}
	mac, err := hex.DecodeString(t.MAC)
	if err != nil {
		return err
	}
	if !hmac.Equal(b, mac) {
		return dnsv2.ErrSig
	}
	return nil
}
