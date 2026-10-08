package syntax

import "strings"

// Newest is the very-count of the newest dialect Parse reads: 0 for the
// original's Sorted!, 1 for Very Sorted! (#25), 2 for Very Very Sorted!
// (#30). There is no version 1.0:
// the dialects are the versions (#39), so sorted --version prints
// Marker(Newest), and the release that brings a dialect is tagged
// Tag(Newest).
const Newest = 2

// Marker is the last sentence of a program in the dialect with n verys:
// "This code is cool.", "This code is very cool.", ...
func Marker(n int) string { return "This code is " + strings.Repeat("very ", n) + "cool." }

// Tag is the release tag of the dialect with n verys: "cool",
// "very-cool", ...
func Tag(n int) string { return strings.Repeat("very-", n) + "cool" }
