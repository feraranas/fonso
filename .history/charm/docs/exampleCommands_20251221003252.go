// Commands in Bubble Tea
// For this tutorial we're building a very simple program that makes an HTTP request to a server and reports the status code of the response.

// We'll import a few necessary packages and put the URL we're going to check in a const.

package docs

import (
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const url = "https://charm.sh/"

// The Model
// Next we'll define our model.
// The only things we need to store are the status code of the HTTP response and a possible error.

type model struct {
	status int
	err    error
}

// Commands and Messages
// Cmds are functions that perform some I/O and then return a Msg.
// Checking the time, ticking a timer, reading from the disk, and network stuff are all I/O and should be run through commands.
// That might sound harsh, but it will keep your Bubble Tea program straightforward and simple.

// Anyway, let's write a Cmd that makes a request to a server and returns the result as a Msg.
func checkServer() tea.Msg {

	// Create an HTTP client and make a GET request.
	c := &http.Client{Timeout: 10 * time.Second}
	res, err := c.Get(url)

	if err != nil {
		// There was an error making our request. Wrap the error we received
		// in a message and return it.
		return errMsg{err}
	}
	// We received a response from the server. Return the HTTP status code
	// as a message.
	return statusMsg(res.StatusCode)
}

type statusMsg int

type errMsg struct{ err error }

// For messages that contain errors it's often handy to also implement the
// error interface on the message.
func (e errMsg) Error() string { return e.err.Error() }
