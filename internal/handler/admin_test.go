package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Сессионный токен не должен совпадать с паролем и быть стабильным.
func TestSessionTokenHidesPassword(t *testing.T) {
	const pw = "s3cret"
	tok := sessionToken(pw)

	if tok == pw {
		t.Fatal("sessionToken() вернул сам пароль")
	}
	if tok == "" {
		t.Fatal("sessionToken() пуст")
	}
	if got := sessionToken(pw); got != tok {
		t.Fatalf("sessionToken() нестабилен: %q != %q", got, tok)
	}
	if sessionToken("other") == tok {
		t.Fatal("разные пароли дали одинаковый токен")
	}
}

func TestCheckAuth(t *testing.T) {
	h := &AdminHandler{password: "s3cret", sessionToken: sessionToken("s3cret")}

	cases := []struct {
		name   string
		cookie string
		set    bool
		want   bool
	}{
		{"верный токен", sessionToken("s3cret"), true, true},
		{"сырой пароль", "s3cret", true, false},
		{"чужой токен", "deadbeef", true, false},
		{"куки нет", "", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/admin/dashboard", nil)
			if tc.set {
				r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: tc.cookie})
			}
			if got := h.checkAuth(r); got != tc.want {
				t.Fatalf("checkAuth() = %v, want %v", got, tc.want)
			}
		})
	}
}
