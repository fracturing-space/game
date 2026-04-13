// Package scene owns scene lifecycle command and event behavior.
//
// Scene command-level authorization currently stays in the admission catalog
// because the active rules are coarse AI-GM checks rather than resource-level
// decisions inside Decide.
package scene
