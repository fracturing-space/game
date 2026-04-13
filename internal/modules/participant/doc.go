// Package participant implements the engine module for participant
// behavior.
//
// It exists to decide participant commands into events and fold those events
// into campaign state. This is where campaign membership behavior lives after
// command validation and request admission have already happened.
//
// Read internal/participant first if you need the participant contract or
// state shape. Read this package when you need to change how joining and
// participant state updates behave.
package participant
