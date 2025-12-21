// For this tutorial, we're making a shopping list.

// To start we'll define our package and import some libraries.
// Our only external import will be the Bubble Tea library, which we'll call tea for short.

package main

// These imports will be used later on the tutorial. If you save the file
// now, Go might complain they are unused, but that's fine.
// You may also need to run `go mod tidy` to download bubbletea and its
// dependencies.
import (
	tea "github.com/charmbracelet/bubbletea"
)

// Bubble Tea programs are comprised of a model that describes the application state and three simple methods on that model:

//////////// Init, a function that returns an initial command for the application to run.
//////////// Update, a function that handles incoming events and updates the model accordingly.
//////////// View, a function that renders the UI based on the data in the model.

// The Model
// So let's start by defining our model which will store our application's state. It can be any type, but a struct usually makes the most sense.
type model struct {
	choices  []string         // items on the to-do list
	cursor   int              // which to-do list item our cursor is pointing at
	selected map[int]struct{} // which to-do items are selected
}

// Initialization
// Next, we’ll define our application’s initial state. In this case, we’re defining a function to return our initial model,
// however, we could just as easily define the initial model as a variable elsewhere, too.
func initialModel() model {
	return model{
		// Our to-do list is a grocery list
		choices: []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
	}
}

// Next, we define the Init method.
// Init can return a Cmd that could perform some initial I/O.
// For now, we don't need to do any I/O, so for the command, we'll just return nil, which translates to "no command."
func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

// The Update Method
// Next up is the update method.
// The update function is called when ”things happen.”
// Its job is to look at what has happened and return an updated model in response.
// It can also return a Cmd to make more things happen, but for now don't worry about that part.

// In our case, when a user presses the down arrow, Update’s job is to notice that the down arrow was pressed and move the cursor accordingly (or not).
//
// The “something happened” comes in the form of a Msg, which can be any type.
//
// Messages are the result of some I/O that took place, such as a keypress, timer tick, or a response from a server.
//
// We usually figure out which type of Msg we received with a type switch, but you could also use a type assertion.
//
// For now, we'll just deal with tea.KeyMsg messages, which are automatically sent to the update function when keys are pressed.
