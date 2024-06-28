package ocua

import (
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

func getSessionCookie(cookies []playwright.Cookie) (playwright.Cookie, bool) {
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie.Name, "SSESS") {
			return cookie, true
		}
	}
	return playwright.Cookie{}, false
}

func getCookieExpires(cookie playwright.Cookie) time.Time {
	expires := time.Unix(int64(cookie.Expires), 0)
	return expires
}

// ClientSessionRefresher runs in the background and replaces main client cookies
// before they expire. minimizes time where the main client is unusable.
type ClientSessionRefresher struct {
	playwright.Browser
	playwright.BrowserNewContextOptions
	*Client

	Username string
	Password string

	Logger *slog.Logger
}

func (refresher *ClientSessionRefresher) RunOnce() (time.Time, error) {
	cookies, expires, err := refresher.getFreshCookies()
	if err != nil {
		return time.Time{}, err
	}

	err = refresher.Client.setCookies(cookies)
	if err != nil {
		return time.Time{}, err
	}

	return expires, nil
}

func (refresher *ClientSessionRefresher) RunBackground() {
	once := sync.Once{}
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		for {
			// refresh the session
			cookies, expires, err := refresher.getFreshCookies()
			if err != nil {
				refresher.Logger.Error("failed to refresh cookies", "error", err)
				time.Sleep(time.Minute * 30)
				continue
			}

			// refresh the client with new cookies
			err = refresher.Client.setCookies(cookies)
			if err != nil {
				refresher.Logger.Error("failed to refresh client cookies", "error", err)
				time.Sleep(time.Minute * 30)
				continue
			}

			// successfully refreshed cookies
			refresher.Logger.Info("successfully refreshed cookies")

			once.Do(func() {
				wg.Done()
			})

			// sleep until the next refresh
			d := time.Until(expires.Add(-24 * time.Hour))
			time.Sleep(d)
		}
	}()

	wg.Wait()
}

func (refresher *ClientSessionRefresher) getFreshCookies() ([]playwright.Cookie, time.Time, error) {
	// launch a new browser context with no cookies
	context, err := refresher.NewContext(refresher.BrowserNewContextOptions)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer context.Close()

	// login to OCUA this would result in cookies saved to the browser
	err = Login(refresher.Username, refresher.Password, context)
	if err != nil {
		return nil, time.Time{}, err
	}

	// get cookies
	cookies, err := context.Cookies()
	if err != nil {
		return nil, time.Time{}, err
	}

	// get the session cookie
	cookie, ok := getSessionCookie(cookies)
	if !ok {
		return nil, time.Time{}, errors.New("could not find session cookie")
	}

	// get the expiration time of the cookie
	expires := getCookieExpires(cookie)

	return cookies, expires, err
}

type Client struct {
	sync.RWMutex
	playwright.BrowserContext
}

func (client *Client) setCookies(cookies []playwright.Cookie) error {
	// convert cookies to optional
	optional := make([]playwright.OptionalCookie, len(cookies))
	for i, cookie := range cookies {
		optional[i] = cookie.ToOptionalCookie()
	}

	// acquire write lock
	client.Lock()
	defer client.Unlock()

	// reset cookies
	err := client.ClearCookies(playwright.BrowserContextClearCookiesOptions{})
	if err != nil {
		return err
	}

	return client.AddCookies(optional)
}

func (client *Client) GetTeam(teamID string) (map[string]Player, error) {
	client.RLock()
	defer client.RUnlock()

	page, err := GetTeamPage(teamID, client.BrowserContext)
	if err != nil {
		return nil, err
	}

	attendance, err := ParseTeamPage(page)
	if err != nil {
		return nil, err
	}

	return attendance, nil
}

func (client *Client) GetAttendance(teamID string) ([]Attendance, error) {
	client.RLock()
	defer client.RUnlock()

	page, err := GetAttendancePage(teamID, client.BrowserContext)
	if err != nil {
		return nil, err
	}

	attendance, err := ParseAttendancePage(page)
	if err != nil {
		return nil, err
	}

	return attendance, nil
}
