package externalurl

import (
	"errors"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
)

const MaxURLBytes = 2048

var ErrInvalidURL = errors.New("Only an absolute HTTPS URL with an ASCII host and no credentials is allowed.")
var ErrBusy = errors.New("An external-link confirmation is already open.")

// Validate uses a conservative ASCII URI boundary to avoid shell/browser parser differences.
// International hosts must use punycode and non-ASCII paths/queries must be percent encoded.
func Validate(raw string) (string, error) {
	if len(raw) == 0 || len(raw) > MaxURLBytes {
		return "", ErrInvalidURL
	}
	for _, c := range raw {
		if c <= 0x20 || c >= 0x7f || strings.ContainsRune(`\"<>`+"`", c) {
			return "", ErrInvalidURL
		}
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return "", ErrInvalidURL
	}
	for _, c := range decoded {
		if c < 0x20 || c == 0x7f || c == '\\' {
			return "", ErrInvalidURL
		}
	}
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.Opaque != "" || u.User != nil || u.Host == "" || strings.Contains(u.Host, "%") {
		return "", ErrInvalidURL
	}
	host := u.Hostname()
	if strings.Contains(host, ":") {
		address, err := netip.ParseAddr(host)
		if err != nil || !address.Is6() || address.Zone() != "" {
			return "", ErrInvalidURL
		}
	} else {
		if strings.ContainsAny(u.Host, "[]") {
			return "", ErrInvalidURL
		}
		domain := strings.TrimSuffix(host, ".")
		if len(domain) == 0 || len(domain) > 253 {
			return "", ErrInvalidURL
		}
		for _, label := range strings.Split(domain, ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return "", ErrInvalidURL
			}
			for _, c := range label {
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
					return "", ErrInvalidURL
				}
			}
		}
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", ErrInvalidURL
		}
	} else if strings.HasSuffix(u.Host, ":") {
		return "", ErrInvalidURL
	}
	u.Scheme, u.Host = "https", strings.ToLower(u.Host)
	canonical := u.String()
	if len(canonical) > MaxURLBytes {
		return "", ErrInvalidURL
	}
	return canonical, nil
}

type Opener struct {
	Confirm   func(string) (bool, error)
	Launch    func(string) error
	IsClosing func() bool
	busy      atomic.Bool
}

func (o *Opener) Open(raw string) (bool, error) {
	target, err := Validate(raw)
	if err != nil {
		return false, err
	}
	if !o.busy.CompareAndSwap(false, true) {
		return false, ErrBusy
	}
	defer o.busy.Store(false)
	if o.Confirm == nil || o.Launch == nil {
		return false, errors.New("external browser is unavailable")
	}
	approved, err := o.Confirm(target)
	if err != nil || !approved {
		return false, err
	}
	if o.IsClosing != nil && o.IsClosing() {
		return false, nil
	}
	if err := o.Launch(target); err != nil {
		return false, err
	}
	return true, nil
}
