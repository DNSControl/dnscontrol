package soautil

import "strings"

// RFC5322MailToBind converts a user@host email address to BIND format.
func RFC5322MailToBind(rfc5322Mail string) string {
	res := strings.SplitN(rfc5322Mail, "@", 2)
	user, domain := res[0], res[1]
	// RFC-1035 [Section-8]
	user = strings.ReplaceAll(user, ".", "\\.")
	return user + "." + domain
}

// BindMailToRFC5322 converts a BIND format mailbox to a user@host email address.
// It returns "" if there is no unescaped dot to split on.
func BindMailToRFC5322(bindMail string) string {
	bindMail = strings.TrimRight(bindMail, ".")
	for i := 0; i < len(bindMail); i++ {
		if bindMail[i] == '\\' {
			i++
			continue
		}
		if bindMail[i] != '.' {
			continue
		}
		user, domain := bindMail[:i], bindMail[i+1:]
		if user == "" {
			return ""
		}
		// RFC-1035 [Section-8]
		return strings.ReplaceAll(user, "\\.", ".") + "@" + domain
	}
	return ""
}
