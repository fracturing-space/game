// Package engine assembles the runtime registry and catalogs that route
// validated commands and events to the packages that own their behavior.
//
// It exists to keep orchestration generic while letting domain areas register
// their own command and event specs. The engine knows which module owns a type,
// but it does not define the rules for that type itself.
//
// Contributors usually visit this package when adding or wiring a new module,
// or when tracing how a validated command reaches domain behavior through the
// assembled engine artifacts. The behavior itself lives in internal/modules/....
package engine
