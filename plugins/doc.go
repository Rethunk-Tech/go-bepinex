// Package plugins reads which BepInEx plugins the compiled assemblies in a folder declare, through go-dotnet, and
// orders them the way BepInEx 5's chainloader loads them. A plugin DLL is untrusted input: it is only parsed, never
// loaded or run.
package plugins
