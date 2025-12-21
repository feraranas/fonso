// For this tutorial, we're making a shopping list.

// To start we'll define our package and import some libraries. Our only external import will be the Bubble Tea library, which we'll call tea for short.

package main

// These imports will be used later on the tutorial. If you save the file
// now, Go might complain they are unused, but that's fine.
// You may also need to run `go mod tidy` to download bubbletea and its
// dependencies.

// Bubble Tea programs are comprised of a model that describes the application state and three simple methods on that model:

// Init, a function that returns an initial command for the application to run.
// Update, a function that handles incoming events and updates the model accordingly.
// View, a function that renders the UI based on the data in the model.
// The Model
// So let's start by defining our model which will store our application's state. It can be any type, but a struct usually makes the most sense.
