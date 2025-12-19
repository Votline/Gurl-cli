package cookies

import (
	"os"
	"net/url"
	"net/http"
	"encoding/json"
	"net/http/cookiejar"

	"go.uber.org/zap"
)

type CookiesClient struct {
	log *zap.Logger
	jar http.CookieJar
	cookies map[string][]*http.Cookie
	ckPath string
}
func NewClient(ckPath string, log *zap.Logger) *CookiesClient {
	ckCl := CookiesClient{log: log, ckPath: ckPath}
	if err := ckCl.loadCookie(); err != nil {
		log.Error("Failed to load cookie", zap.Error(err))
	}
	return &ckCl
}

func (cc *CookiesClient) SaveCookies() error {
	cc.log.Info("Saving cookies with", zap.Any("cookies", cc.cookies))

	file, err := os.Create(cc.ckPath)
	if err != nil {
		cc.log.Error("Couldn't create file for cookies", zap.Error(err))
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	return encoder.Encode(cc.cookies)
}

func (cc *CookiesClient) loadCookie() error {
	if cc.ckPath == "" {return nil}

	data, err := os.ReadFile(cc.ckPath)
	if err != nil {
		cc.log.Error("Couldn't load cookies file", zap.Error(err))
		return err
	}
	
	loadedCoookies := make(map[string][]*http.Cookie)
	if err := json.Unmarshal(data, &loadedCoookies); err != nil {
		cc.log.Error("Decode cookies error", zap.Error(err))
		return err
	}

	cc.cookies = loadedCoookies
	return cc.loadedToJar()
}

func (cc *CookiesClient) loadedToJar() error {
	if cc.jar == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			cc.log.Error("Create cookie's jar error", zap.Error(err))
			return err
		}
		cc.jar = jar
	}

	for dom, ck := range cc.cookies {
		url, err := url.Parse("http://"+dom)
		urls, err1 := url.Parse("https://"+dom)
		if err != nil || err1 != nil {
			continue
		}
		cc.jar.SetCookies(url, ck)
		cc.jar.SetCookies(urls, ck)
	}

	return nil
}

func (cc *CookiesClient) UpdateCookies(u *url.URL) {
	if cc.jar == nil || cc.cookies == nil { return }
	cc.cookies[u.Host] = cc.jar.Cookies(u)
}

func (cc *CookiesClient) GetJar() http.CookieJar {
	if cc == nil {
		return nil
	}
	if cc.jar == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			cc.log.Error("Failed to create new jar", zap.Error(err))
		}
		cc.jar = jar
	}
	return cc.jar
}
func (cc *CookiesClient) GetCookies() map[string][]*http.Cookie {
	if cc.cookies == nil {
		cc.cookies = make(map[string][]*http.Cookie)
	}
	return cc.cookies
}
func (cc *CookiesClient) SetCookies(cks map[string][]*http.Cookie) {
	cc.cookies = cks
}
