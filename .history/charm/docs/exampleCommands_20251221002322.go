// Commands in Bubble Tea
// For this tutorial we're building a very simple program that makes an HTTP request to a server and reports the status code of the response.

// We'll import a few necessary packages and put the URL we're going to check in a const.

package docs

const url = "https://charm.sh/"

// The Model
// Next we'll define our model.
// The only things we need to store are the status code of the HTTP response and a possible error.

type model struct {
	status int
	err    error
}
