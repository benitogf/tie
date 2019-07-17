package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/benitogf/samo"
	"golang.org/x/crypto/bcrypt"
)

// User :
type User struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Account  string `json:"account"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Credentials :
type Credentials struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Token    string `json:"token"`
	Role     string `json:"role"`
}

// TokenAuth :
type TokenAuth struct {
	tokenStore          *JwtStore
	store               samo.Database
	getter              TokenGetter
	UnauthorizedHandler http.HandlerFunc
}

// TokenGetter :
type TokenGetter interface {
	GetTokenFromRequest(req *http.Request) string
}

// Token :
type Token interface {
	IsExpired() bool
	fmt.Stringer
	ClaimGetter
}

// ClaimSetter :
type ClaimSetter interface {
	SetClaim(string, interface{}) ClaimSetter
}

// ClaimGetter :
type ClaimGetter interface {
	Claims(string) interface{}
}

// BearerGetter :
type BearerGetter struct {
	Header string
}

var (
	userRegexp  = regexp.MustCompile("^[a-zA-Z0-9_]{2,15}$")
	emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	phoneRegexp = regexp.MustCompile("^[0-9_-]{6,15}$")
	roles       = map[string]string{"admin": "admin"}
)

// DefaultUnauthorizedHandler :
func DefaultUnauthorizedHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprint(w, "unauthorized")
}

// GetTokenFromRequest :
func (b *BearerGetter) GetTokenFromRequest(req *http.Request) string {
	// log.Println("header:", req.Header)
	authStr := req.Header.Get(b.Header)
	if !strings.HasPrefix(authStr, "Bearer ") {
		return ""
	}

	return authStr[7:]
}

// NewHeaderBearerTokenGetter :
func NewHeaderBearerTokenGetter(header string) *BearerGetter {
	return &BearerGetter{
		Header: header,
	}
}

// NewTokenAuth :
// Returns a TokenAuth object implemting Handler interface
// if a handler is given it proxies the request to the handler
// if a unauthorizedHandler is provided, unauthorized requests will be handled by this HandlerFunc,
// otherwise a default unauthorized handler is used.
// store is the TokenStore that stores and verify the tokens
func NewTokenAuth(tokenStore *JwtStore, store samo.Database) *TokenAuth {
	t := &TokenAuth{
		tokenStore: tokenStore,
		store:      store,
	}
	t.getter = NewHeaderBearerTokenGetter("Authorization")
	t.UnauthorizedHandler = DefaultUnauthorizedHandler
	return t
}

// Verify : wrap a HandlerFunc to be authenticated
func (t *TokenAuth) Verify(req *http.Request) bool {
	_, err := t.Authenticate(req)
	if err != nil {
		return false
	}
	// context.Set(req, "token", token)
	return true
}

// Authenticate :
func (t *TokenAuth) Authenticate(r *http.Request) (Token, error) {
	strToken := t.getter.GetTokenFromRequest(r)
	if strToken == "" {
		return nil, errors.New("token required")
	}
	token, err := t.tokenStore.CheckToken(strToken)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// Authorize method
func (t *TokenAuth) getUser(account string) (User, error) {
	var u User
	raw, err := t.store.Get("sa", "users/"+account)
	if err != nil {
		return u, err
	}
	var obj samo.Object
	err = json.Unmarshal(raw, &obj)
	if err != nil {
		return u, err
	}
	err = json.Unmarshal([]byte(obj.Data), &u)
	if err != nil {
		return u, err
	}
	return u, nil
}

func getCredentials(r *http.Request) (Credentials, error) {
	dec := json.NewDecoder(r.Body)
	var c Credentials
	err := dec.Decode(&c)
	if err != nil {
		return c, err
	}

	return c, nil
}

func (t *TokenAuth) checkCredentials(c Credentials) (User, error) {
	u, err := t.getUser(c.Account)
	if err != nil {
		return u, errors.New("user not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(c.Password))
	if err != nil {
		return u, errors.New("wrong password")
	}

	return u, nil
}

// Profile returns to the client the correspondent user profile for the token provided
func (t *TokenAuth) Profile(w http.ResponseWriter, r *http.Request) {
	token, err := t.Authenticate(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%s", errors.New("this request is not authorized"))
		return
	}
	switch r.Method {
	case "GET":
		u, err := t.getUser(token.Claims("iss").(string))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "bad token, couldnt find the issuer profile")
			return
		}
		u.Password = ""
		role, otherRole := roles[u.Account]
		if otherRole {
			u.Role = role
		}
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		enc.Encode(&u)
		return
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Method not suported")
		return
	}
}

// Authorize will claim a token on POST and refresh the claim on PUT
func (t *TokenAuth) Authorize(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	c, err := getCredentials(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	switch r.Method {
	case "POST":
		_, err = t.checkCredentials(c)
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, err.Error())
			return
		}
		break
	case "PUT":
		_, err = t.getUser(c.Account)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err.Error())
			return
		}
		if c.Token != "" {
			oldToken, err := t.tokenStore.CheckToken(c.Token)
			if err == nil {
				w.WriteHeader(http.StatusNotModified)
				fmt.Fprint(w, errors.New("token not expired"))
				return
			}

			if oldToken.Claims("iss").(string) != c.Account {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, errors.New("token doesn't match the account"))
				return
			}

			if err.Error() != "Token expired" {
				w.WriteHeader(http.StatusNotModified)
				fmt.Fprint(w, err)
				return
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, errors.New("empty token"))
			return
		}
		break
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Method not suported")
		return
	}

	newToken := t.tokenStore.NewToken()
	newToken.SetClaim("iss", c.Account)
	c.Password = ""
	c.Role = "user"
	role, otherRole := roles[c.Account]
	if otherRole {
		c.Role = role
	}
	newToken.SetClaim("role", c.Role)
	c.Token = newToken.String()
	w.Header().Add("content-type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(&c)
}

// Register will create a new user
func (t *TokenAuth) Register(w http.ResponseWriter, r *http.Request) {
	var u User
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err := decoder.Decode(&u)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", err)
		return
	}

	if u.Account == "" || u.Name == "" || u.Password == "" || u.Email == "" || u.Phone == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("new user data incomplete"))
		return
	}

	if !userRegexp.MatchString(u.Account) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("account cannot contain special characters, only numbers or lowercase letters and character count must be between 2 and 15"))
		return
	}

	if len(u.Password) < 3 || len(u.Password) > 88 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("password character count must be between 2 and 88"))
		return
	}

	if !userRegexp.MatchString(u.Phone) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("phone cannot contain special characters othen than '-' and character count must be between 6 and 15"))
		return
	}

	if !emailRegexp.MatchString(u.Email) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("invalid email address"))
		return
	}

	_, err = t.getUser(u.Account)

	if err == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%s", errors.New("account name taken"))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.MinCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	u.Password = string(hash)
	u.Role = "user"
	role, otherRole := roles[u.Account]
	if otherRole {
		u.Role = role
	}
	dataBytes := new(bytes.Buffer)
	json.NewEncoder(dataBytes).Encode(u)
	key, index, now := (&samo.Keys{}).Build("mo", "users", u.Account, "r", "/")
	index, err = t.store.Set(key, index, now, string(dataBytes.Bytes()))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	newToken := t.tokenStore.NewToken()
	newToken.SetClaim("iss", u.Account)
	newToken.SetClaim("role", u.Role)
	c := Credentials{
		Account: u.Account,
		Token:   newToken.String(),
	}
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(&c)
}

// Available will check if an account is taken
func (t *TokenAuth) Available(w http.ResponseWriter, r *http.Request) {
	account := r.FormValue("account")
	_, err := t.getUser(account)

	if err == nil {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, "account taken")
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "account available")
}
